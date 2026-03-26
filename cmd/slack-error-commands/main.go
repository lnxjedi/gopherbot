package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

const (
	defaultPhrase      = "No command matched in channel"
	defaultOutputRel   = "~/tmp/slack-error-commands.out"
	defaultPageSize    = 100
	defaultMaxPages    = 20
	defaultHTTPTimeout = 45 * time.Second
)

type config struct {
	outputPath string
	phrase     string
	botNames   []string
	pageSize   int
	maxPages   int
	quiet      bool
}

type reportEntry struct {
	ChannelID      string
	ChannelName    string
	Permalink      string
	ReplyTS        string
	ReplyTime      time.Time
	ReplyText      string
	ReplyUser      string
	ReplyUsername  string
	ReplyBotID     string
	ParentTS       string
	ParentTime     time.Time
	ParentText     string
	ParentUser     string
	Threaded       bool
	SourceQueries  []string
	ThreadFetchErr string
}

type liveReport struct {
	file         *os.File
	writer       *bufio.Writer
	detailHeader bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseFlags()
	if err != nil {
		return err
	}

	token := strings.TrimSpace(os.Getenv("GOPHER_USER_TOKEN"))
	if token == "" {
		return errors.New("GOPHER_USER_TOKEN is not set")
	}

	outputPath, err := expandPath(cfg.outputPath)
	if err != nil {
		return err
	}
	cfg.outputPath = outputPath

	progress(cfg, "starting slack unmatched-command scan")
	progress(cfg, "report path: %s", cfg.outputPath)
	if len(cfg.botNames) > 0 {
		progress(cfg, "target bot names: %s", strings.Join(cfg.botNames, ", "))
	} else {
		progress(cfg, "no bot names configured; using generic search queries only")
	}

	client := slack.New(
		token,
		slack.OptionHTTPClient(&http.Client{Timeout: defaultHTTPTimeout}),
	)

	progress(cfg, "authenticating with Slack")
	auth, err := authTestWithRetry(client)
	if err != nil {
		return fmt.Errorf("slack auth test failed: %w", err)
	}
	progress(cfg, "authenticated as %s in workspace %s", firstNonEmpty(auth.User, auth.UserID, "?"), firstNonEmpty(auth.Team, auth.TeamID, "?"))

	queries := buildSearchQueries(cfg.phrase, cfg.botNames)
	progress(cfg, "prepared %d search queries", len(queries))

	reporter, err := createLiveReport(cfg.outputPath)
	if err != nil {
		return err
	}
	defer reporter.Close()

	if err := reporter.WriteHeader(auth, cfg.phrase, queries); err != nil {
		return err
	}
	progress(cfg, "initialized report file")

	results, err := collectEntries(client, queries, cfg, reporter)
	if err != nil {
		return err
	}
	progress(cfg, "search complete; writing %d entries", len(results))

	if err := reporter.WriteSummary(results); err != nil {
		return err
	}
	if err := reporter.Flush(); err != nil {
		return fmt.Errorf("flush report: %w", err)
	}

	progress(cfg, "report written to %s", outputPath)
	fmt.Printf("wrote %d entries to %s\n", len(results), outputPath)
	return nil
}

func parseFlags() (config, error) {
	defaultOutput, err := expandPath(defaultOutputRel)
	if err != nil {
		return config{}, err
	}

	var botNamesCSV string
	cfg := config{}
	flag.StringVar(&cfg.outputPath, "output", defaultOutput, "report output path")
	flag.StringVar(&cfg.phrase, "phrase", defaultPhrase, "message text to search for")
	flag.StringVar(&botNamesCSV, "botnames", os.Getenv("GOPHER_SLACK_BOT_NAMES"), "comma-separated Slack bot names for targeted from:botname queries")
	flag.IntVar(&cfg.pageSize, "count", defaultPageSize, "search results per page (max 100)")
	flag.IntVar(&cfg.maxPages, "max-pages", defaultMaxPages, "max search pages to request per query")
	flag.BoolVar(&cfg.quiet, "quiet", false, "suppress progress messages on stderr")
	flag.Parse()

	cfg.phrase = strings.TrimSpace(cfg.phrase)
	if cfg.phrase == "" {
		return config{}, errors.New("phrase must not be empty")
	}
	if cfg.pageSize < 1 || cfg.pageSize > 100 {
		return config{}, fmt.Errorf("count must be between 1 and 100, got %d", cfg.pageSize)
	}
	if cfg.maxPages < 1 {
		return config{}, fmt.Errorf("max-pages must be >= 1, got %d", cfg.maxPages)
	}

	cfg.botNames = splitCSV(botNamesCSV)
	return cfg, nil
}

func collectEntries(client *slack.Client, queries []string, cfg config, reporter *liveReport) ([]reportEntry, error) {
	seen := make(map[string]*reportEntry)
	var lastErr error

	for i, query := range queries {
		progress(cfg, "searching query %d/%d: %s", i+1, len(queries), query)
		matches, err := searchAll(client, query, cfg.pageSize, cfg.maxPages, cfg)
		if err != nil {
			progress(cfg, "query failed: %v", err)
			lastErr = err
			continue
		}
		progress(cfg, "query %d/%d returned %d matches", i+1, len(queries), len(matches))

		newCandidates := 0
		for _, match := range matches {
			if !strings.Contains(strings.ToLower(match.Text), strings.ToLower(cfg.phrase)) {
				continue
			}
			key := match.Channel.ID + "\x00" + match.Timestamp
			if existing, ok := seen[key]; ok {
				existing.SourceQueries = appendUnique(existing.SourceQueries, query)
				continue
			}
			newCandidates++
			entry := reportEntry{
				ChannelID:     match.Channel.ID,
				ChannelName:   match.Channel.Name,
				Permalink:     match.Permalink,
				ReplyTS:       match.Timestamp,
				ReplyTime:     parseSlackTimestamp(match.Timestamp),
				ReplyText:     strings.TrimSpace(match.Text),
				ReplyUser:     strings.TrimSpace(match.User),
				ReplyUsername: strings.TrimSpace(match.Username),
				SourceQueries: []string{query},
			}
			if newCandidates == 1 {
				progress(cfg, "fetching thread parents for new matches from this query")
			}
			if newCandidates == 1 || newCandidates%25 == 0 {
				progress(cfg, "hydrating candidate %d for current query", newCandidates)
			}
			hydrated, err := hydrateEntry(client, entry, cfg)
			if err != nil {
				progress(cfg, "thread fetch failed for %s (%s): %v", firstNonEmpty(entry.ChannelName, entry.ChannelID, "?"), entry.ReplyTS, err)
				entry.ThreadFetchErr = err.Error()
				seen[key] = &entry
				if err := reporter.WriteEntry(entry); err != nil {
					return nil, err
				}
				continue
			}
			seen[key] = &hydrated
			if err := reporter.WriteEntry(hydrated); err != nil {
				return nil, err
			}
		}
		progress(cfg, "query %d/%d added %d unique candidate replies", i+1, len(queries), newCandidates)
	}

	if len(seen) == 0 && lastErr != nil {
		return nil, fmt.Errorf("search failed for all queries: %w", lastErr)
	}

	out := make([]reportEntry, 0, len(seen))
	for _, entry := range seen {
		sort.Strings(entry.SourceQueries)
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ReplyTime.After(out[j].ReplyTime)
	})
	if len(out) == 0 {
		progress(cfg, "no matching error replies found")
	} else {
		progress(cfg, "deduped to %d unique replies", len(out))
	}
	return out, nil
}

func hydrateEntry(client *slack.Client, entry reportEntry, cfg config) (reportEntry, error) {
	threadTS := threadRootTimestamp(entry)
	progress(cfg, "using thread root ts %s for reply ts %s", threadTS, entry.ReplyTS)
	msgs, err := getConversationRepliesAll(client, entry.ChannelID, threadTS, cfg)
	if err != nil {
		return entry, err
	}
	if len(msgs) == 0 {
		return entry, errors.New("conversations.replies returned no messages")
	}

	parent := msgs[0]
	entry.ParentTS = parent.Timestamp
	entry.ParentTime = parseSlackTimestamp(parent.Timestamp)
	entry.ParentText = strings.TrimSpace(parent.Text)
	entry.ParentUser = firstNonEmpty(parent.User, parent.Username, parent.BotID)
	entry.Threaded = parent.Timestamp != entry.ReplyTS

	for _, msg := range msgs {
		if msg.Timestamp != entry.ReplyTS {
			continue
		}
		entry.ReplyText = strings.TrimSpace(firstNonEmpty(msg.Text, entry.ReplyText))
		entry.ReplyUser = strings.TrimSpace(firstNonEmpty(msg.User, entry.ReplyUser))
		entry.ReplyUsername = strings.TrimSpace(firstNonEmpty(msg.Username, entry.ReplyUsername))
		entry.ReplyBotID = strings.TrimSpace(msg.BotID)
		if msg.ThreadTimestamp != "" && msg.ThreadTimestamp != msg.Timestamp {
			entry.Threaded = true
		}
		break
	}

	if entry.ParentTS == "" {
		return entry, errors.New("parent timestamp was empty")
	}
	return entry, nil
}

func searchAll(client *slack.Client, query string, pageSize, maxPages int, cfg config) ([]slack.SearchMessage, error) {
	params := slack.NewSearchParameters()
	params.Count = pageSize
	params.Sort = "timestamp"
	params.SortDirection = "desc"

	var out []slack.SearchMessage
	for page := 1; page <= maxPages; page++ {
		params.Page = page
		progress(cfg, "requesting search page %d for query %q", page, query)
		result, err := searchMessagesWithRetry(client, query, params)
		if err != nil {
			return nil, fmt.Errorf("search query %q page %d: %w", query, page, err)
		}
		progress(cfg, "received %d matches on page %d", len(result.Matches), page)
		out = append(out, result.Matches...)

		if len(result.Matches) == 0 {
			break
		}
		if result.Paging.Pages > 0 && page >= result.Paging.Pages {
			break
		}
		if result.Pagination.PageCount > 0 && page >= result.Pagination.PageCount {
			break
		}
	}
	return out, nil
}

func getConversationRepliesAll(client *slack.Client, channelID, ts string, cfg config) ([]slack.Message, error) {
	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: ts,
		Limit:     200,
	}

	var out []slack.Message
	page := 0
	for {
		page++
		progress(cfg, "fetching thread page %d for channel %s ts %s", page, channelID, ts)
		msgs, hasMore, nextCursor, err := getConversationRepliesWithRetry(client, params)
		if err != nil {
			return nil, err
		}
		progress(cfg, "thread page %d returned %d messages", page, len(msgs))
		out = append(out, msgs...)
		if !hasMore && nextCursor == "" {
			break
		}
		params.Cursor = nextCursor
		if params.Cursor == "" {
			break
		}
	}
	return dedupeMessages(out), nil
}

func dedupeMessages(in []slack.Message) []slack.Message {
	seen := make(map[string]struct{}, len(in))
	out := make([]slack.Message, 0, len(in))
	for _, msg := range in {
		if msg.Timestamp == "" {
			continue
		}
		if _, ok := seen[msg.Timestamp]; ok {
			continue
		}
		seen[msg.Timestamp] = struct{}{}
		out = append(out, msg)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp < out[j].Timestamp
	})
	return out
}

func formatSummary(entries []reportEntry) string {
	var b strings.Builder
	uniqueCommands := countParentTexts(entries)
	threadFailures := 0
	threadedCount := 0
	for _, entry := range entries {
		if entry.Threaded {
			threadedCount++
		}
		if entry.ThreadFetchErr != "" {
			threadFailures++
		}
	}

	fmt.Fprintf(&b, "Summary:\n")
	fmt.Fprintf(&b, "- Unique error replies: %d\n", len(entries))
	fmt.Fprintf(&b, "- Thread replies confirmed: %d\n", threadedCount)
	fmt.Fprintf(&b, "- Thread fetch failures: %d\n", threadFailures)
	fmt.Fprintf(&b, "- Unique original messages: %d\n", len(uniqueCommands))
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "Top original messages:\n")
	top := summarizeCounts(uniqueCommands)
	if len(top) == 0 {
		fmt.Fprintf(&b, "- none\n")
	} else {
		for _, item := range top {
			fmt.Fprintf(&b, "- %d x %s\n", item.Count, item.Text)
		}
	}
	fmt.Fprintf(&b, "\n")
	return b.String()
}

type summaryItem struct {
	Text  string
	Count int
}

func countParentTexts(entries []reportEntry) map[string]int {
	out := make(map[string]int)
	for _, entry := range entries {
		text := normalizeText(entry.ParentText)
		if text == "" {
			continue
		}
		out[text]++
	}
	return out
}

func summarizeCounts(counts map[string]int) []summaryItem {
	items := make([]summaryItem, 0, len(counts))
	for text, count := range counts {
		items = append(items, summaryItem{Text: text, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Text < items[j].Text
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > 20 {
		items = items[:20]
	}
	return items
}

func buildSearchQueries(phrase string, botNames []string) []string {
	quoted := strconv.Quote(phrase)
	queries := make([]string, 0, len(botNames)*2+2)
	for _, botName := range botNames {
		queries = append(queries,
			fmt.Sprintf("%s has:thread from:%s", quoted, botName),
			fmt.Sprintf("%s from:%s", quoted, botName),
		)
	}
	queries = append(queries,
		fmt.Sprintf("%s has:thread", quoted),
		quoted,
	)
	return uniqueStrings(queries)
}

func expandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path must not be empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path), nil
}

func splitCSV(in string) []string {
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func appendUnique(dst []string, value string) []string {
	for _, item := range dst {
		if item == value {
			return dst
		}
	}
	return append(dst, value)
}

func normalizeText(in string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(in)), " ")
}

func indentBlock(in string) string {
	lines := strings.Split(in, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func formatTime(t time.Time, raw string) string {
	if t.IsZero() {
		return raw
	}
	return t.Format(time.RFC3339) + " (" + raw + ")"
}

func parseSlackTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	parts := strings.SplitN(ts, ".", 2)
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}
	}
	var nsec int64
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > 9 {
			fraction = fraction[:9]
		}
		for len(fraction) < 9 {
			fraction += "0"
		}
		nsec, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return time.Time{}
		}
	}
	return time.Unix(sec, nsec).UTC()
}

func threadRootTimestamp(entry reportEntry) string {
	if ts := threadTimestampFromPermalink(entry.Permalink); ts != "" {
		return ts
	}
	return entry.ReplyTS
}

func threadTimestampFromPermalink(permalink string) string {
	if strings.TrimSpace(permalink) == "" {
		return ""
	}
	u, err := url.Parse(permalink)
	if err != nil {
		return ""
	}
	threadTS := strings.TrimSpace(u.Query().Get("thread_ts"))
	if threadTS != "" {
		return threadTS
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func authTestWithRetry(client *slack.Client) (*slack.AuthTestResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := client.AuthTest()
		if err == nil {
			return resp, nil
		}
		if !sleepIfRateLimited(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func searchMessagesWithRetry(client *slack.Client, query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := client.SearchMessages(query, params)
		if err == nil {
			return resp, nil
		}
		if !sleepIfRateLimited(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func getConversationRepliesWithRetry(client *slack.Client, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		msgs, hasMore, nextCursor, err := client.GetConversationReplies(params)
		if err == nil {
			return msgs, hasMore, nextCursor, nil
		}
		if !sleepIfRateLimited(err) {
			return nil, false, "", err
		}
		lastErr = err
	}
	return nil, false, "", lastErr
}

func sleepIfRateLimited(err error) bool {
	var rateLimitErr *slack.RateLimitedError
	if !errors.As(err, &rateLimitErr) {
		return false
	}
	fmt.Fprintf(os.Stderr, "[%s] rate limited by Slack; sleeping for %s\n", time.Now().Format(time.RFC3339), rateLimitErr.RetryAfter)
	time.Sleep(rateLimitErr.RetryAfter)
	return true
}

func progress(cfg config, format string, args ...any) {
	if cfg.quiet {
		return
	}
	fmt.Fprintf(os.Stderr, "[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func createLiveReport(path string) (*liveReport, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create report file: %w", err)
	}
	return &liveReport{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (r *liveReport) WriteHeader(auth *slack.AuthTestResponse, phrase string, queries []string) error {
	now := time.Now().Format(time.RFC3339)
	if _, err := fmt.Fprintf(r.writer, "Slack unmatched-command report\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Generated: %s\n", now); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Workspace: %s (%s)\n", firstNonEmpty(auth.Team, "?"), firstNonEmpty(auth.TeamID, "?")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Authenticated user: %s (%s)\n", firstNonEmpty(auth.User, "?"), firstNonEmpty(auth.UserID, "?")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Search phrase: %q\n", phrase); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Status: in progress\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Queries tried:\n"); err != nil {
		return err
	}
	for _, query := range queries {
		if _, err := fmt.Fprintf(r.writer, "- %s\n", query); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(r.writer, "\nEntries will be appended below as they are found.\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "\n"); err != nil {
		return err
	}
	return r.writer.Flush()
}

func (r *liveReport) WriteEntry(entry reportEntry) error {
	if !r.detailHeader {
		if _, err := fmt.Fprintf(r.writer, "Details:\n"); err != nil {
			return err
		}
		r.detailHeader = true
	}
	if _, err := fmt.Fprintf(r.writer, "--------------------------------------------------------------------------------\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Channel: #%s (%s)\n", firstNonEmpty(entry.ChannelName, "?"), firstNonEmpty(entry.ChannelID, "?")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Reply time: %s\n", formatTime(entry.ReplyTime, entry.ReplyTS)); err != nil {
		return err
	}
	if entry.ParentTS != "" {
		if _, err := fmt.Fprintf(r.writer, "Parent time: %s\n", formatTime(entry.ParentTime, entry.ParentTS)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(r.writer, "Reply user: %s\n", firstNonEmpty(entry.ReplyUsername, entry.ReplyUser, entry.ReplyBotID, "?")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Threaded: %t\n", entry.Threaded); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Permalink: %s\n", firstNonEmpty(entry.Permalink, "?")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Queries: %s\n", strings.Join(entry.SourceQueries, " | ")); err != nil {
		return err
	}
	if entry.ThreadFetchErr != "" {
		if _, err := fmt.Fprintf(r.writer, "Thread fetch error: %s\n", entry.ThreadFetchErr); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(r.writer, "Original message:\n%s\n", indentBlock(firstNonEmpty(entry.ParentText, "<unavailable>"))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Error reply:\n%s\n", indentBlock(firstNonEmpty(entry.ReplyText, "<unavailable>"))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "\n"); err != nil {
		return err
	}
	return r.writer.Flush()
}

func (r *liveReport) WriteSummary(entries []reportEntry) error {
	if _, err := fmt.Fprintf(r.writer, "================================================================================\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Status: complete\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.writer, "Completed: %s\n\n", time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	if _, err := fmt.Fprint(r.writer, formatSummary(entries)); err != nil {
		return err
	}
	return nil
}

func (r *liveReport) Flush() error {
	return r.writer.Flush()
}

func (r *liveReport) Close() error {
	if err := r.writer.Flush(); err != nil {
		_ = r.file.Close()
		return err
	}
	return r.file.Close()
}
