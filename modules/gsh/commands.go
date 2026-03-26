package gsh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/lnxjedi/gopherbot/robot"
	ucore "github.com/u-root/u-root/pkg/core"
	ubase64 "github.com/u-root/u-root/pkg/core/base64"
	ucat "github.com/u-root/u-root/pkg/core/cat"
	uchmod "github.com/u-root/u-root/pkg/core/chmod"
	ucp "github.com/u-root/u-root/pkg/core/cp"
	ufind "github.com/u-root/u-root/pkg/core/find"
	ugzip "github.com/u-root/u-root/pkg/core/gzip"
	uls "github.com/u-root/u-root/pkg/core/ls"
	umkdir "github.com/u-root/u-root/pkg/core/mkdir"
	umv "github.com/u-root/u-root/pkg/core/mv"
	urm "github.com/u-root/u-root/pkg/core/rm"
	ushasum "github.com/u-root/u-root/pkg/core/shasum"
	utar "github.com/u-root/u-root/pkg/core/tar"
	utouch "github.com/u-root/u-root/pkg/core/touch"
	uxargs "github.com/u-root/u-root/pkg/core/xargs"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

type commandHandler func(context.Context, []string) error

type setWorkingDirectoryAPI interface {
	SetWorkingDirectory(string) bool
}

func (c *shellContext) execHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if len(args) == 0 {
			return next(ctx, args)
		}
		if handler, ok := c.commandMap()[normalizeCommandName(args[0])]; ok {
			return handler(ctx, args[1:])
		}
		return next(ctx, args)
	}
}

func (c *shellContext) commandMap() map[string]commandHandler {
	return map[string]commandHandler{
		"say":                             c.cmdSay,
		"saythread":                       c.cmdSayThread,
		"reply":                           c.cmdReply,
		"replythread":                     c.cmdReplyThread,
		"sendchannelmessage":              c.cmdSendChannelMessage,
		"sendchannelthreadmessage":        c.cmdSendChannelThreadMessage,
		"sendusermessage":                 c.cmdSendUserMessage,
		"senduserchannelmessage":          c.cmdSendUserChannelMessage,
		"senduserchannelthreadmessage":    c.cmdSendUserChannelThreadMessage,
		"sendprotocoluserchannelmessage":  c.cmdSendProtocolUserChannelMessage,
		"promptforreply":                  c.cmdPromptForReply,
		"promptthreadforreply":            c.cmdPromptThreadForReply,
		"promptuserforreply":              c.cmdPromptUserForReply,
		"promptuserchannelforreply":       c.cmdPromptUserChannelForReply,
		"promptuserchannelthreadforreply": c.cmdPromptUserChannelThreadForReply,
		"checkadmin":                      c.cmdCheckAdmin,
		"subscribe":                       c.cmdSubscribe,
		"unsubscribe":                     c.cmdUnsubscribe,
		"remember":                        c.cmdRemember,
		"rememberthread":                  c.cmdRememberThread,
		"remembercontext":                 c.cmdRememberContext,
		"remembercontextthread":           c.cmdRememberContextThread,
		"recall":                          c.cmdRecall,
		"deletememory":                    c.cmdDeleteMemory,
		"getparameter":                    c.cmdGetParameter,
		"getoauth2token":                  c.cmdGetOAuth2Token,
		"linkoauth2user":                  c.cmdLinkOAuth2User,
		"unlinkoauth2user":                c.cmdUnlinkOAuth2User,
		"setparameter":                    c.cmdSetParameter,
		"setworkingdirectory":             c.cmdSetWorkingDirectory,
		"addtask":                         c.cmdAddTask,
		"finaltask":                       c.cmdFinalTask,
		"failtask":                        c.cmdFailTask,
		"addjob":                          c.cmdAddJob,
		"spawnjob":                        c.cmdSpawnJob,
		"addcommand":                      c.cmdAddCommand,
		"finalcommand":                    c.cmdFinalCommand,
		"failcommand":                     c.cmdFailCommand,
		"exclusive":                       c.cmdExclusive,
		"elevate":                         c.cmdElevate,
		"getbotattribute":                 c.cmdGetBotAttribute,
		"getsenderattribute":              c.cmdGetSenderAttribute,
		"getuserattribute":                c.cmdGetUserAttribute,
		"log":                             c.cmdLog,
		"gettaskconfig":                   c.cmdGetTaskConfig,
		"pause":                           c.cmdPause,
		"cat":                             c.wrapCore(ucat.New),
		"base64":                          c.wrapCore(ubase64.New),
		"chmod":                           c.wrapCore(uchmod.New),
		"cp":                              c.wrapCore(ucp.New),
		"find":                            c.wrapCore(ufind.New),
		"gzip":                            c.wrapNamedCore(func() ucore.Command { return ugzip.New("gzip") }),
		"gunzip":                          c.wrapNamedCore(func() ucore.Command { return ugzip.New("gunzip") }),
		"gzcat":                           c.wrapNamedCore(func() ucore.Command { return ugzip.New("gzcat") }),
		"ls":                              c.wrapCore(uls.New),
		"mkdir":                           c.wrapCore(umkdir.New),
		"mktemp":                          c.cmdMktemp,
		"mv":                              c.wrapCore(umv.New),
		"rm":                              c.wrapCore(urm.New),
		"shasum":                          c.wrapCore(ushasum.New),
		"tar":                             c.wrapCore(utar.New),
		"touch":                           c.wrapCore(utouch.New),
		"xargs":                           c.wrapCore(uxargs.New),
		"sha1sum":                         c.wrapArgPrefix(ushasum.New, "-a", "1"),
		"sha256sum":                       c.wrapArgPrefix(ushasum.New, "-a", "256"),
		"sha512sum":                       c.wrapArgPrefix(ushasum.New, "-a", "512"),
		"basename":                        c.cmdBasename,
		"dirname":                         c.cmdDirname,
		"pwd":                             c.cmdPwd,
		"env":                             c.cmdEnv,
		"which":                           c.cmdWhich,
		"sleep":                           c.cmdSleep,
		"seq":                             c.cmdSeq,
		"yes":                             c.cmdYes,
		"head":                            c.cmdHead,
		"tail":                            c.cmdTail,
		"wc":                              c.cmdWc,
		"tr":                              c.cmdTr,
		"sort":                            c.cmdSort,
		"uniq":                            c.cmdUniq,
		"grep":                            c.cmdGrep,
		"jq":                              c.cmdJq,
		"realpath":                        c.cmdRealpath,
		"date":                            c.cmdDate,
	}
}

func normalizeCommandName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("-", "", "_", "").Replace(name)
	return strings.ToLower(name)
}

func (c *shellContext) wrapCore(factory func() ucore.Command) commandHandler {
	return c.wrapNamedCore(factory)
}

func (c *shellContext) wrapNamedCore(factory func() ucore.Command) commandHandler {
	return func(ctx context.Context, args []string) error {
		return c.runCoreCommand(ctx, factory(), args)
	}
}

func (c *shellContext) wrapArgPrefix(factory func() ucore.Command, prefix ...string) commandHandler {
	return func(ctx context.Context, args []string) error {
		combined := append(append([]string(nil), prefix...), args...)
		return c.runCoreCommand(ctx, factory(), combined)
	}
}

func (c *shellContext) runCoreCommand(ctx context.Context, cmd ucore.Command, args []string) error {
	hc := interp.HandlerCtx(ctx)
	cmd.SetIO(hc.Stdin, hc.Stdout, hc.Stderr)
	cmd.SetWorkingDir(hc.Dir)
	cmd.SetLookupEnv(func(key string) (string, bool) {
		vr := hc.Env.Get(key)
		if !vr.IsSet() {
			return "", false
		}
		return vr.String(), true
	})
	return cmd.RunContext(ctx, args...)
}

func (c *shellContext) cmdPause(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "Pause requires exactly one duration in seconds")
	}
	seconds, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return usageError(ctx, "Pause requires a numeric duration in seconds")
	}
	if c.bot == nil {
		time.Sleep(time.Duration(seconds * float64(time.Second)))
		return nil
	}
	c.bot.Pause(seconds)
	return nil
}

func (c *shellContext) cmdSay(ctx context.Context, args []string) error {
	bot, msg, err := c.botWithMessageOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	return retToError(bot.Say(msg))
}

func (c *shellContext) cmdSayThread(ctx context.Context, args []string) error {
	bot, msg, err := c.botWithMessageOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	return retToError(bot.SayThread(msg))
}

func (c *shellContext) cmdReply(ctx context.Context, args []string) error {
	bot, msg, err := c.botWithMessageOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	return retToError(bot.Reply(msg))
}

func (c *shellContext) cmdReplyThread(ctx context.Context, args []string) error {
	bot, msg, err := c.botWithMessageOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	return retToError(bot.ReplyThread(msg))
}

func (c *shellContext) cmdSendChannelMessage(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 2 {
		return usageError(ctx, "SendChannelMessage requires channel and message")
	}
	return retToError(bot.SendChannelMessage(rest[0], strings.Join(rest[1:], " ")))
}

func (c *shellContext) cmdSendChannelThreadMessage(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 3 {
		return usageError(ctx, "SendChannelThreadMessage requires channel, thread, and message")
	}
	return retToError(bot.SendChannelThreadMessage(rest[0], rest[1], strings.Join(rest[2:], " ")))
}

func (c *shellContext) cmdSendUserMessage(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 2 {
		return usageError(ctx, "SendUserMessage requires user and message")
	}
	return retToError(bot.SendUserMessage(rest[0], strings.Join(rest[1:], " ")))
}

func (c *shellContext) cmdSendUserChannelMessage(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 3 {
		return usageError(ctx, "SendUserChannelMessage requires user, channel, and message")
	}
	return retToError(bot.SendUserChannelMessage(rest[0], rest[1], strings.Join(rest[2:], " ")))
}

func (c *shellContext) cmdSendUserChannelThreadMessage(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 4 {
		return usageError(ctx, "SendUserChannelThreadMessage requires user, channel, thread, and message")
	}
	return retToError(bot.SendUserChannelThreadMessage(rest[0], rest[1], rest[2], strings.Join(rest[3:], " ")))
}

func (c *shellContext) cmdSendProtocolUserChannelMessage(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 4 {
		return usageError(ctx, "SendProtocolUserChannelMessage requires protocol, user, channel, and message")
	}
	return retToError(bot.SendProtocolUserChannelMessage(rest[0], rest[1], rest[2], strings.Join(rest[3:], " ")))
}

func (c *shellContext) cmdCheckAdmin(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return usageError(ctx, "CheckAdmin does not take arguments")
	}
	if c.bot == nil {
		return usageError(ctx, "CheckAdmin is unavailable during configure")
	}
	hc := interp.HandlerCtx(ctx)
	value := c.bot.CheckAdmin()
	fmt.Fprintln(hc.Stdout, strconv.FormatBool(value))
	if value {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdSubscribe(ctx context.Context, args []string) error {
	return c.booleanRobotCommand(ctx, args, "Subscribe", c.bot.Subscribe)
}

func (c *shellContext) cmdUnsubscribe(ctx context.Context, args []string) error {
	return c.booleanRobotCommand(ctx, args, "Unsubscribe", c.bot.Unsubscribe)
}

func (c *shellContext) cmdRemember(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return usageError(ctx, "Remember requires key and value")
	}
	c.bot.Remember(args[0], args[1], len(args) > 2 && parseTruthy(args[2]))
	return nil
}

func (c *shellContext) cmdRememberThread(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return usageError(ctx, "RememberThread requires key and value")
	}
	c.bot.RememberThread(args[0], args[1], len(args) > 2 && parseTruthy(args[2]))
	return nil
}

func (c *shellContext) cmdRememberContext(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return usageError(ctx, "RememberContext requires context and value")
	}
	c.bot.RememberContext(args[0], args[1])
	return nil
}

func (c *shellContext) cmdRememberContextThread(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return usageError(ctx, "RememberContextThread requires context and value")
	}
	c.bot.RememberContextThread(args[0], args[1])
	return nil
}

func (c *shellContext) cmdRecall(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return usageError(ctx, "Recall requires key")
	}
	shared := len(args) > 1 && parseTruthy(args[1])
	value := c.bot.Recall(args[0], shared)
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, value)
	return nil
}

func (c *shellContext) cmdDeleteMemory(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return usageError(ctx, "DeleteMemory requires key")
	}
	shared := len(args) > 1 && parseTruthy(args[1])
	c.bot.DeleteMemory(args[0], shared)
	return nil
}

func (c *shellContext) cmdGetParameter(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "GetParameter requires parameter name")
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, c.bot.GetParameter(args[0]))
	return nil
}

func (c *shellContext) cmdGetOAuth2Token(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return usageError(ctx, "GetOAuth2Token requires provider and user")
	}
	token, ret := c.bot.GetOAuth2Token(args[0], args[1])
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, token)
	return retCodeError(ret)
}

func (c *shellContext) cmdLinkOAuth2User(ctx context.Context, args []string) error {
	if len(args) < 3 || len(args) > 6 {
		return usageError(ctx, "LinkOAuth2User requires provider, user, accessToken, and optional refreshToken, expiresIn, tokenType")
	}
	link := &robot.OAuth2LinkRequest{
		Provider:    args[0],
		User:        args[1],
		AccessToken: args[2],
		TokenType:   "Bearer",
	}
	if len(args) > 3 {
		link.RefreshToken = args[3]
	}
	if len(args) > 4 {
		expiresIn, err := strconv.Atoi(args[4])
		if err != nil {
			return usageError(ctx, "LinkOAuth2User expiresIn must be numeric")
		}
		link.ExpiresIn = expiresIn
	}
	if len(args) > 5 && args[5] != "" {
		link.TokenType = args[5]
	}
	return retCodeError(c.bot.LinkOAuth2User(link))
}

func (c *shellContext) cmdUnlinkOAuth2User(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return usageError(ctx, "UnlinkOAuth2User requires provider and user")
	}
	return retCodeError(c.bot.UnlinkOAuth2User(args[0], args[1]))
}

func (c *shellContext) cmdSetParameter(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return usageError(ctx, "SetParameter requires name and value")
	}
	if c.bot.SetParameter(args[0], args[1]) {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdSetWorkingDirectory(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "SetWorkingDirectory requires a path")
	}
	wdAPI, ok := any(c.bot).(setWorkingDirectoryAPI)
	if !ok {
		return usageError(ctx, "SetWorkingDirectory is unavailable in this context")
	}
	if wdAPI.SetWorkingDirectory(args[0]) {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdAddTask(ctx context.Context, args []string) error {
	return c.pipeCommand(ctx, args, "AddTask", c.bot.AddTask)
}

func (c *shellContext) cmdFinalTask(ctx context.Context, args []string) error {
	return c.pipeCommand(ctx, args, "FinalTask", c.bot.FinalTask)
}

func (c *shellContext) cmdFailTask(ctx context.Context, args []string) error {
	return c.pipeCommand(ctx, args, "FailTask", c.bot.FailTask)
}

func (c *shellContext) cmdAddJob(ctx context.Context, args []string) error {
	return c.pipeCommand(ctx, args, "AddJob", c.bot.AddJob)
}

func (c *shellContext) cmdSpawnJob(ctx context.Context, args []string) error {
	return c.pipeCommand(ctx, args, "SpawnJob", c.bot.SpawnJob)
}

func (c *shellContext) cmdAddCommand(ctx context.Context, args []string) error {
	return c.pipelineCommand(ctx, args, "AddCommand", c.bot.AddCommand)
}

func (c *shellContext) cmdFinalCommand(ctx context.Context, args []string) error {
	return c.pipelineCommand(ctx, args, "FinalCommand", c.bot.FinalCommand)
}

func (c *shellContext) cmdFailCommand(ctx context.Context, args []string) error {
	return c.pipelineCommand(ctx, args, "FailCommand", c.bot.FailCommand)
}

func (c *shellContext) cmdExclusive(ctx context.Context, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return usageError(ctx, "Exclusive requires tag and optional queueTask boolean")
	}
	queue := false
	if len(args) == 2 {
		queue = parseTruthy(args[1])
	}
	if c.bot.Exclusive(args[0], queue) {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdElevate(ctx context.Context, args []string) error {
	immediate := false
	if len(args) > 1 {
		return usageError(ctx, "Elevate takes at most one boolean argument")
	}
	if len(args) == 1 {
		immediate = parseTruthy(args[0])
	}
	if c.bot.Elevate(immediate) {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdGetBotAttribute(ctx context.Context, args []string) error {
	return c.attrCommand(ctx, args, "GetBotAttribute", func() *robot.AttrRet {
		return c.bot.GetBotAttribute(args[0])
	})
}

func (c *shellContext) cmdGetSenderAttribute(ctx context.Context, args []string) error {
	return c.attrCommand(ctx, args, "GetSenderAttribute", func() *robot.AttrRet {
		return c.bot.GetSenderAttribute(args[0])
	})
}

func (c *shellContext) cmdGetUserAttribute(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return usageError(ctx, "GetUserAttribute requires user and attribute")
	}
	attr := c.bot.GetUserAttribute(args[0], args[1])
	if attr == nil {
		return interp.ExitStatus(uint8(robot.AttributeNotFound))
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, attr.Attribute)
	if attr.RetVal == robot.Ok {
		return nil
	}
	return interp.ExitStatus(uint8(attr.RetVal))
}

func (c *shellContext) cmdLog(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return usageError(ctx, "Log requires a numeric level and message")
	}
	level, err := strconv.Atoi(args[0])
	if err != nil {
		return usageError(ctx, "Log requires a numeric level")
	}
	if c.bot.Log(robot.LogLevel(level), strings.Join(args[1:], " ")) {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdGetTaskConfig(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return usageError(ctx, "GetTaskConfig does not take arguments")
	}
	var mapCfg map[string]interface{}
	ret := c.bot.GetTaskConfig(&mapCfg)
	if ret == robot.Ok {
		return writeJSON(ctx, mapCfg)
	}
	if ret == robot.ConfigUnmarshalError {
		var sliceCfg []interface{}
		ret = c.bot.GetTaskConfig(&sliceCfg)
		if ret == robot.Ok {
			return writeJSON(ctx, sliceCfg)
		}
	}
	return interp.ExitStatus(uint8(ret))
}

func (c *shellContext) cmdPromptForReply(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 2 {
		return usageError(ctx, "PromptForReply requires regexID and prompt")
	}
	reply, ret := c.promptRetry(func() (string, robot.RetVal) {
		return bot.PromptForReply(rest[0], strings.Join(rest[1:], " "))
	})
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, reply)
	return retCodeError(ret)
}

func (c *shellContext) cmdPromptThreadForReply(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 2 {
		return usageError(ctx, "PromptThreadForReply requires regexID and prompt")
	}
	reply, ret := c.promptRetry(func() (string, robot.RetVal) {
		return bot.PromptThreadForReply(rest[0], strings.Join(rest[1:], " "))
	})
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, reply)
	return retCodeError(ret)
}

func (c *shellContext) cmdPromptUserForReply(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 3 {
		return usageError(ctx, "PromptUserForReply requires regexID, user, and prompt")
	}
	reply, ret := c.promptRetry(func() (string, robot.RetVal) {
		return bot.PromptUserForReply(rest[0], rest[1], strings.Join(rest[2:], " "))
	})
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, reply)
	return retCodeError(ret)
}

func (c *shellContext) cmdPromptUserChannelForReply(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 4 {
		return usageError(ctx, "PromptUserChannelForReply requires regexID, user, channel, and prompt")
	}
	reply, ret := c.promptRetry(func() (string, robot.RetVal) {
		return bot.PromptUserChannelForReply(rest[0], rest[1], rest[2], strings.Join(rest[3:], " "))
	})
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, reply)
	return retCodeError(ret)
}

func (c *shellContext) cmdPromptUserChannelThreadForReply(ctx context.Context, args []string) error {
	bot, rest, err := c.botWithOptions(ctx, args, false, false)
	if err != nil {
		return err
	}
	if len(rest) < 5 {
		return usageError(ctx, "PromptUserChannelThreadForReply requires regexID, user, channel, thread, and prompt")
	}
	reply, ret := c.promptRetry(func() (string, robot.RetVal) {
		return bot.PromptUserChannelThreadForReply(rest[0], rest[1], rest[2], rest[3], strings.Join(rest[4:], " "))
	})
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, reply)
	return retCodeError(ret)
}

func (c *shellContext) cmdBasename(ctx context.Context, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return usageError(ctx, "basename requires path and optional suffix")
	}
	base := filepath.Base(args[0])
	if len(args) == 2 && strings.HasSuffix(base, args[1]) {
		base = strings.TrimSuffix(base, args[1])
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, base+"\n")
	return nil
}

func (c *shellContext) cmdDirname(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "dirname requires exactly one path")
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, filepath.Dir(args[0])+"\n")
	return nil
}

func (c *shellContext) cmdPwd(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return usageError(ctx, "pwd does not take arguments")
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, interp.HandlerCtx(ctx).Dir+"\n")
	return nil
}

func (c *shellContext) cmdEnv(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return usageError(ctx, "env does not take arguments in gsh")
	}
	hc := interp.HandlerCtx(ctx)
	lines := []string{}
	hc.Env.Each(func(name string, vr expand.Variable) bool {
		if vr.IsSet() {
			lines = append(lines, fmt.Sprintf("%s=%s", name, vr.String()))
		}
		return true
	})
	sort.Strings(lines)
	for _, line := range lines {
		fmt.Fprintln(hc.Stdout, line)
	}
	return nil
}

func (c *shellContext) cmdWhich(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "which requires exactly one command")
	}
	hc := interp.HandlerCtx(ctx)
	if _, ok := c.commandMap()[normalizeCommandName(args[0])]; ok {
		fmt.Fprintln(hc.Stdout, args[0]+" (gsh builtin)")
		return nil
	}
	path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
	if err != nil {
		return interp.ExitStatus(1)
	}
	fmt.Fprintln(hc.Stdout, path)
	return nil
}

func (c *shellContext) cmdSleep(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "sleep requires exactly one duration in seconds")
	}
	seconds, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return usageError(ctx, "sleep requires a numeric duration")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(seconds * float64(time.Second))):
		return nil
	}
}

func (c *shellContext) cmdMktemp(ctx context.Context, args []string) error {
	dirMode := false
	quiet := false
	rest := args
	for len(rest) > 0 && strings.HasPrefix(rest[0], "-") && rest[0] != "-" {
		switch rest[0] {
		case "-d":
			dirMode = true
		case "-q":
			quiet = true
		default:
			return usageError(ctx, "mktemp only supports -d and -q in gsh")
		}
		rest = rest[1:]
	}
	if len(rest) > 1 {
		return usageError(ctx, "mktemp takes at most one template")
	}
	template := "tmp.XXXXXXXXXX"
	if len(rest) == 1 {
		template = rest[0]
	}
	hc := interp.HandlerCtx(ctx)
	absTemplate := resolvePath(hc.Dir, template)
	dir := filepath.Dir(absTemplate)
	pattern := filepath.Base(absTemplate)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if quiet {
			return interp.ExitStatus(1)
		}
		return err
	}
	pattern = normalizeMktempPattern(pattern)
	var path string
	if dirMode {
		created, err := os.MkdirTemp(dir, pattern)
		if err != nil {
			if quiet {
				return interp.ExitStatus(1)
			}
			return err
		}
		path = created
	} else {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			if quiet {
				return interp.ExitStatus(1)
			}
			return err
		}
		_ = file.Close()
		path = file.Name()
	}
	fmt.Fprintln(hc.Stdout, path)
	return nil
}

func (c *shellContext) cmdSeq(ctx context.Context, args []string) error {
	if len(args) < 1 || len(args) > 3 {
		return usageError(ctx, "seq requires 1, 2, or 3 numeric arguments")
	}
	values := make([]float64, len(args))
	for i, arg := range args {
		v, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return usageError(ctx, "seq arguments must be numeric")
		}
		values[i] = v
	}
	first, increment, last := 1.0, 1.0, values[0]
	if len(values) == 2 {
		first, last = values[0], values[1]
	}
	if len(values) == 3 {
		first, increment, last = values[0], values[1], values[2]
	}
	if increment == 0 {
		return usageError(ctx, "seq increment must not be zero")
	}
	for current := first; ; current += increment {
		if (increment > 0 && current > last) || (increment < 0 && current < last) {
			break
		}
		if current == float64(int64(current)) {
			fmt.Fprintf(interp.HandlerCtx(ctx).Stdout, "%d\n", int64(current))
		} else {
			fmt.Fprintf(interp.HandlerCtx(ctx).Stdout, "%g\n", current)
		}
	}
	return nil
}

func (c *shellContext) cmdYes(ctx context.Context, args []string) error {
	word := "y"
	if len(args) > 0 {
		word = strings.Join(args, " ")
	}
	hc := interp.HandlerCtx(ctx)
	for {
		if _, err := io.WriteString(hc.Stdout, word+"\n"); err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (c *shellContext) cmdHead(ctx context.Context, args []string) error {
	count, files, err := parseCountFlag(args, 10)
	if err != nil {
		return usageError(ctx, err.Error())
	}
	readers, err := inputReaders(interp.HandlerCtx(ctx), files)
	if err != nil {
		return err
	}
	defer closeReaders(readers)
	for _, reader := range readers {
		scanner := bufio.NewScanner(reader.reader)
		for i := 0; i < count && scanner.Scan(); i++ {
			fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (c *shellContext) cmdTail(ctx context.Context, args []string) error {
	count, files, err := parseCountFlag(args, 10)
	if err != nil {
		return usageError(ctx, err.Error())
	}
	readers, err := inputReaders(interp.HandlerCtx(ctx), files)
	if err != nil {
		return err
	}
	defer closeReaders(readers)
	for _, reader := range readers {
		lines, err := readLines(reader.reader)
		if err != nil {
			return err
		}
		start := len(lines) - count
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, line)
		}
	}
	return nil
}

func (c *shellContext) cmdWc(ctx context.Context, args []string) error {
	lineOnly := false
	files := args
	if len(args) > 0 && args[0] == "-l" {
		lineOnly = true
		files = args[1:]
	}
	readers, err := inputReaders(interp.HandlerCtx(ctx), files)
	if err != nil {
		return err
	}
	defer closeReaders(readers)
	hc := interp.HandlerCtx(ctx)
	for _, reader := range readers {
		data, err := io.ReadAll(reader.reader)
		if err != nil {
			return err
		}
		lines := 0
		if len(data) > 0 {
			lines = bytes.Count(data, []byte("\n"))
			if data[len(data)-1] != '\n' {
				lines++
			}
		}
		if lineOnly {
			fmt.Fprintf(hc.Stdout, "%d", lines)
		} else {
			words := len(strings.Fields(string(data)))
			fmt.Fprintf(hc.Stdout, "%d %d %d", lines, words, len(data))
		}
		if reader.name != "" {
			fmt.Fprintf(hc.Stdout, " %s", reader.name)
		}
		fmt.Fprintln(hc.Stdout)
	}
	return nil
}

func (c *shellContext) cmdTr(ctx context.Context, args []string) error {
	deleteMode := false
	if len(args) < 2 {
		return usageError(ctx, "tr requires source and destination sets or -d set")
	}
	if args[0] == "-d" {
		deleteMode = true
		args = args[1:]
		if len(args) != 1 {
			return usageError(ctx, "tr -d requires exactly one set")
		}
	}
	hc := interp.HandlerCtx(ctx)
	data, err := io.ReadAll(hc.Stdin)
	if err != nil {
		return err
	}
	text := string(data)
	if deleteMode {
		for _, ch := range parseTrSet(args[0]) {
			text = strings.ReplaceAll(text, string(ch), "")
		}
		_, _ = io.WriteString(hc.Stdout, text)
		return nil
	}
	src, dst := parseTrSet(args[0]), parseTrSet(args[1])
	if len(src) == 0 || len(dst) == 0 {
		_, _ = io.WriteString(hc.Stdout, text)
		return nil
	}
	replacer := make([]string, 0, len(src)*2)
	for i, ch := range src {
		mapped := string(dst[min(i, len(dst)-1)])
		replacer = append(replacer, string(ch), mapped)
	}
	_, _ = io.WriteString(hc.Stdout, strings.NewReplacer(replacer...).Replace(text))
	return nil
}

func (c *shellContext) cmdSort(ctx context.Context, args []string) error {
	reverse := false
	if len(args) > 0 && args[0] == "-r" {
		reverse = true
		args = args[1:]
	}
	readers, err := inputReaders(interp.HandlerCtx(ctx), args)
	if err != nil {
		return err
	}
	defer closeReaders(readers)
	all := []string{}
	for _, reader := range readers {
		lines, err := readLines(reader.reader)
		if err != nil {
			return err
		}
		all = append(all, lines...)
	}
	sort.Strings(all)
	if reverse {
		for i := len(all) - 1; i >= 0; i-- {
			fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, all[i])
		}
		return nil
	}
	for _, line := range all {
		fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, line)
	}
	return nil
}

func (c *shellContext) cmdUniq(ctx context.Context, args []string) error {
	counts := false
	if len(args) > 0 && args[0] == "-c" {
		counts = true
		args = args[1:]
	}
	readers, err := inputReaders(interp.HandlerCtx(ctx), args)
	if err != nil {
		return err
	}
	defer closeReaders(readers)
	for _, reader := range readers {
		lines, err := readLines(reader.reader)
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			continue
		}
		current := lines[0]
		currentCount := 1
		flush := func() {
			if counts {
				fmt.Fprintf(interp.HandlerCtx(ctx).Stdout, "%d %s\n", currentCount, current)
			} else {
				fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, current)
			}
		}
		for _, line := range lines[1:] {
			if line == current {
				currentCount++
				continue
			}
			flush()
			current = line
			currentCount = 1
		}
		flush()
	}
	return nil
}

func (c *shellContext) cmdGrep(ctx context.Context, args []string) error {
	flags, rest, err := parseGrepFlags(args)
	if err != nil {
		return usageError(ctx, err.Error())
	}
	if len(rest) < 1 {
		return usageError(ctx, "grep requires a pattern")
	}
	pattern := rest[0]
	files := rest[1:]
	if flags.ignoreCase && !strings.HasPrefix(pattern, "(?i)") {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	readers, err := inputReaders(interp.HandlerCtx(ctx), files)
	if err != nil {
		return err
	}
	defer closeReaders(readers)
	matchedAny := false
	hc := interp.HandlerCtx(ctx)
	for _, reader := range readers {
		scanner := bufio.NewScanner(reader.reader)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			matched := re.MatchString(line)
			if flags.invert {
				matched = !matched
			}
			if !matched {
				continue
			}
			matchedAny = true
			if flags.quiet {
				return nil
			}
			prefix := ""
			if len(readers) > 1 && reader.name != "" {
				prefix = reader.name + ":"
			}
			if flags.lineNumber {
				prefix += strconv.Itoa(lineNo) + ":"
			}
			fmt.Fprintln(hc.Stdout, prefix+line)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	if matchedAny {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) cmdRealpath(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError(ctx, "realpath requires exactly one path")
	}
	path, err := filepath.Abs(resolvePath(interp.HandlerCtx(ctx).Dir, args[0]))
	if err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = real
	}
	fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, path)
	return nil
}

func (c *shellContext) cmdDate(ctx context.Context, args []string) error {
	now := time.Now()
	layout := time.RFC3339
	if len(args) == 1 && strings.HasPrefix(args[0], "+") {
		layout = translateDateFormat(args[0][1:])
	} else if len(args) > 0 {
		return usageError(ctx, "date only supports +FORMAT in gsh")
	}
	fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, now.Format(layout))
	return nil
}

func (c *shellContext) cmdJq(ctx context.Context, args []string) error {
	rawOutput := false
	compact := false
	nullInput := false
	rest := args
	for len(rest) > 0 && strings.HasPrefix(rest[0], "-") && rest[0] != "-" {
		flag := rest[0]
		if flag == "--" {
			rest = rest[1:]
			break
		}
		for _, ch := range flag[1:] {
			switch ch {
			case 'r':
				rawOutput = true
			case 'c':
				compact = true
			case 'n':
				nullInput = true
			default:
				return usageError(ctx, "jq only supports -r, -c, and -n in gsh")
			}
		}
		rest = rest[1:]
	}
	if len(rest) < 1 {
		return usageError(ctx, "jq requires a query")
	}
	query, err := gojq.Parse(rest[0])
	if err != nil {
		return err
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return err
	}
	inputs, err := jqInputs(interp.HandlerCtx(ctx), rest[1:], nullInput)
	if err != nil {
		return err
	}
	hc := interp.HandlerCtx(ctx)
	enc := json.NewEncoder(hc.Stdout)
	if !compact {
		enc.SetIndent("", "  ")
	}
	for _, input := range inputs {
		iter := code.RunWithContext(ctx, input)
		for {
			value, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := value.(error); ok {
				return err
			}
			if rawOutput {
				if s, ok := value.(string); ok {
					fmt.Fprintln(hc.Stdout, s)
					continue
				}
			}
			if err := enc.Encode(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *shellContext) botWithMessageOptions(ctx context.Context, args []string, direct, threaded bool) (BotAPI, string, error) {
	bot, rest, err := c.botWithOptions(ctx, args, direct, threaded)
	if err != nil {
		return nil, "", err
	}
	if len(rest) == 0 {
		return nil, "", usageError(ctx, "message text is required")
	}
	return bot, strings.Join(rest, " "), nil
}

func (c *shellContext) botWithOptions(ctx context.Context, args []string, direct, threaded bool) (BotAPI, []string, error) {
	if c.bot == nil {
		return nil, nil, usageError(ctx, "robot methods are unavailable during configure")
	}
	format, rest := parseFormatOption(args)
	if format == nil {
		format = defaultFormatFromEnv(c.envMap["GBOT_MESSAGE_FORMAT"])
	}
	bot := c.bot
	if format != nil {
		bot = bot.MessageFormat(*format)
	}
	if direct {
		bot = bot.Direct()
	}
	if threaded {
		bot = bot.Threaded()
	}
	return bot, rest, nil
}

func parseFormatOption(args []string) (*robot.MessageFormat, []string) {
	if len(args) == 0 {
		return nil, args
	}
	switch args[0] {
	case "-f":
		f := robot.Fixed
		return &f, args[1:]
	case "-r":
		f := robot.Raw
		return &f, args[1:]
	case "-v":
		f := robot.Variable
		return &f, args[1:]
	case "-m", "-b":
		f := robot.BasicMarkdown
		return &f, args[1:]
	default:
		return nil, args
	}
}

func defaultFormatFromEnv(value string) *robot.MessageFormat {
	switch strings.TrimSpace(value) {
	case "Fixed":
		f := robot.Fixed
		return &f
	case "Raw":
		f := robot.Raw
		return &f
	case "Variable":
		f := robot.Variable
		return &f
	case "BasicMarkdown":
		f := robot.BasicMarkdown
		return &f
	default:
		return nil
	}
}

func usageError(ctx context.Context, msg string) error {
	hc := interp.HandlerCtx(ctx)
	fmt.Fprintln(hc.Stderr, msg)
	return interp.ExitStatus(2)
}

func retToError(ret robot.RetVal) error {
	if ret == robot.Ok {
		return nil
	}
	return interp.ExitStatus(uint8(ret))
}

func retCodeError(ret robot.RetVal) error {
	if ret == robot.Ok {
		return nil
	}
	if ret == robot.RetryPrompt {
		return interp.ExitStatus(uint8(robot.Interrupted))
	}
	return interp.ExitStatus(uint8(ret))
}

func parseTruthy(value string) bool {
	v, err := strconv.ParseBool(value)
	if err == nil {
		return v
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func writeJSON(ctx context.Context, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, string(data))
	return nil
}

func (c *shellContext) booleanRobotCommand(ctx context.Context, args []string, name string, fn func() bool) error {
	if len(args) != 0 {
		return usageError(ctx, name+" does not take arguments")
	}
	if c.bot == nil {
		return usageError(ctx, name+" is unavailable during configure")
	}
	value := fn()
	fmt.Fprintln(interp.HandlerCtx(ctx).Stdout, strconv.FormatBool(value))
	if value {
		return nil
	}
	return interp.ExitStatus(1)
}

func (c *shellContext) pipeCommand(ctx context.Context, args []string, name string, fn func(string, ...string) robot.RetVal) error {
	if len(args) < 1 {
		return usageError(ctx, name+" requires a task or job name")
	}
	return retToError(fn(args[0], args[1:]...))
}

func (c *shellContext) pipelineCommand(ctx context.Context, args []string, name string, fn func(string, string) robot.RetVal) error {
	if len(args) != 2 {
		return usageError(ctx, name+" requires plugin and command")
	}
	return retToError(fn(args[0], args[1]))
}

func (c *shellContext) attrCommand(ctx context.Context, args []string, name string, fn func() *robot.AttrRet) error {
	if len(args) != 1 {
		return usageError(ctx, name+" requires exactly one attribute")
	}
	attr := fn()
	if attr == nil {
		return interp.ExitStatus(uint8(robot.AttributeNotFound))
	}
	_, _ = io.WriteString(interp.HandlerCtx(ctx).Stdout, attr.Attribute)
	if attr.RetVal == robot.Ok {
		return nil
	}
	return interp.ExitStatus(uint8(attr.RetVal))
}

func (c *shellContext) promptRetry(fn func() (string, robot.RetVal)) (string, robot.RetVal) {
	var reply string
	var ret robot.RetVal
	for i := 0; i < 3; i++ {
		reply, ret = fn()
		if ret != robot.RetryPrompt {
			return reply, ret
		}
	}
	if ret == robot.RetryPrompt {
		return reply, robot.Interrupted
	}
	return reply, ret
}

type namedReader struct {
	name   string
	reader io.ReadCloser
}

func inputReaders(hc interp.HandlerContext, files []string) ([]namedReader, error) {
	if len(files) == 0 {
		return []namedReader{{reader: io.NopCloser(hc.Stdin)}}, nil
	}
	readers := make([]namedReader, 0, len(files))
	for _, file := range files {
		if file == "-" {
			readers = append(readers, namedReader{reader: io.NopCloser(hc.Stdin)})
			continue
		}
		path := resolvePath(hc.Dir, file)
		f, err := os.Open(path)
		if err != nil {
			closeReaders(readers)
			return nil, err
		}
		readers = append(readers, namedReader{name: file, reader: f})
	}
	return readers, nil
}

func closeReaders(readers []namedReader) {
	for _, reader := range readers {
		_ = reader.reader.Close()
	}
}

func resolvePath(dir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}

func readLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func normalizeMktempPattern(pattern string) string {
	for _, token := range []string{"XXXXXXXXXX", "XXXXXXXXX", "XXXXXXXX", "XXXXXXX", "XXXXXX"} {
		if strings.Contains(pattern, token) {
			return strings.Replace(pattern, token, "*", 1)
		}
	}
	return pattern + "*"
}

func parseTrSet(value string) []rune {
	runes := make([]rune, 0, len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			runes = append(runes, rune(value[i]))
			continue
		}
		i++
		switch value[i] {
		case 'n':
			runes = append(runes, '\n')
		case 'r':
			runes = append(runes, '\r')
		case 't':
			runes = append(runes, '\t')
		case '\\':
			runes = append(runes, '\\')
		default:
			runes = append(runes, rune(value[i]))
		}
	}
	return runes
}

func parseCountFlag(args []string, def int) (int, []string, error) {
	if len(args) >= 2 && args[0] == "-n" {
		n, err := strconv.Atoi(args[1])
		if err != nil {
			return 0, nil, fmt.Errorf("invalid -n value")
		}
		return n, args[2:], nil
	}
	return def, args, nil
}

func jqInputs(hc interp.HandlerContext, files []string, nullInput bool) ([]interface{}, error) {
	if nullInput {
		return []interface{}{nil}, nil
	}
	readers, err := inputReaders(hc, files)
	if err != nil {
		return nil, err
	}
	defer closeReaders(readers)
	inputs := []interface{}{}
	for _, reader := range readers {
		dec := json.NewDecoder(reader.reader)
		for {
			var value interface{}
			if err := dec.Decode(&value); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			inputs = append(inputs, value)
		}
	}
	if len(inputs) == 0 {
		return []interface{}{nil}, nil
	}
	return inputs, nil
}

type grepFlags struct {
	quiet      bool
	ignoreCase bool
	invert     bool
	lineNumber bool
}

func parseGrepFlags(args []string) (grepFlags, []string, error) {
	var flags grepFlags
	rest := args
	for len(rest) > 0 && strings.HasPrefix(rest[0], "-") && rest[0] != "-" {
		flag := rest[0]
		rest = rest[1:]
		for _, ch := range strings.TrimPrefix(flag, "-") {
			switch ch {
			case 'q':
				flags.quiet = true
			case 'i':
				flags.ignoreCase = true
			case 'v':
				flags.invert = true
			case 'n':
				flags.lineNumber = true
			case 'E', 'P':
				// Go's regexp engine is our single regex backend here.
			default:
				return grepFlags{}, nil, fmt.Errorf("unsupported grep flag -%c", ch)
			}
		}
	}
	return flags, rest, nil
}

func translateDateFormat(format string) string {
	replacer := strings.NewReplacer(
		"%Y", "2006",
		"%m", "01",
		"%d", "02",
		"%H", "15",
		"%M", "04",
		"%S", "05",
		"%a", "Mon",
		"%b", "Jan",
		"%T", "15:04:05",
		"%F", "2006-01-02",
	)
	return replacer.Replace(format)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
