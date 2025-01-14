#!/usr/bin/env ruby

# Function to escape strings for Go
def escape_go_string(str)
  str.gsub('\\', '\\\\').gsub('"', '\"')
end

# Function to parse INCOMING and OUTGOING lines
def parse_fields(content)
  fields = []
  current = ""
  in_quotes = false
  escape_next = false

  content.each_char do |c|
    if escape_next
      current << c
      escape_next = false
    elsif c == '\\'
      current << c
      escape_next = true
    elsif c == '"'
      in_quotes = !in_quotes
      current << c
    elsif c == ',' && !in_quotes
      fields << current.strip
      current = ""
    else
      current << c
    end
  end
  fields << current.strip unless current.empty?
  fields
end

# Initialize variables
test_cases = []
current_test = nil

ARGF.each_line do |line|
  # Step 1: Check if the line contains "Info: TEST/"
  next unless line.include?("Info: TEST/")

  # Step 2: Strip everything up to and including "Info: TEST/"
  stripped_line = line.split("Info: TEST/").last.strip

  # Step 3: Identify the line type and content
  if stripped_line.start_with?("INCOMING:")
    # Handle INCOMING
    content = stripped_line.sub(/^INCOMING:\s*/, '')
    fields = parse_fields(content)

    if fields.size < 4
      warn "Skipping malformed INCOMING line: #{line}"
      next
    end

    # If there's an existing test case, save it before starting a new one
    if current_test
      test_cases << current_test
    end

    # Initialize a new test case
    current_test = {
      user: fields[0],
      channel: fields[1],
      message: fields[2].gsub(/\A"(.*)"\z/, '\1'),
      threaded: fields[3].downcase == 'true',
      replies: [],
      events: [],
      pause: 0
    }

  elsif stripped_line.start_with?("OUTGOING:")
    # Handle OUTGOING
    next unless current_test # Skip if there's no current test

    content = stripped_line.sub(/^OUTGOING:\s*/, '')
    # Remove surrounding braces if present
    content = content[1..-2] if content.start_with?("{") && content.end_with?("}")

    fields = parse_fields(content)

    if fields.size < 4
      warn "Skipping malformed OUTGOING line: #{line}"
      next
    end

    outgoing_user = fields[0]
    outgoing_channel = fields[1]
    outgoing_message = fields[2].gsub(/\A"(.*)"\z/, '\1')
    outgoing_threaded = fields[3].downcase == 'true'

    # Format user field: null without quotes, others without quotes
    if outgoing_user.downcase == 'null'
      outgoing_user_formatted = 'null'
    else
      outgoing_user_formatted = "#{escape_go_string(outgoing_user)}"
    end

    # Escape message
    outgoing_message_escaped = escape_go_string(outgoing_message)

    # Create TestMessage struct
    test_message = "{#{outgoing_user_formatted}, #{escape_go_string(outgoing_channel)}, \"#{outgoing_message_escaped}\", #{outgoing_threaded}}"
    current_test[:replies] << test_message

  elsif stripped_line.start_with?("EVENTS:")
    # Handle EVENTS
    next unless current_test # Skip if there's no current test

    content = stripped_line.sub(/^EVENTS:\s*/, '')
    # Remove surrounding []Event{...} if present
    content = content[8..-2].strip if content.start_with?("[]Event{") && content.end_with?("}")

    if content.empty?
      current_test[:events] = []
    else
      # Split events by comma and trim
      events = content.split(',').map(&:strip)
      current_test[:events] = events
    end
  else
    # Unknown TEST type, skip
    next
  end
end

# After processing all lines, add the last test case if it exists
test_cases << current_test if current_test

# Step 4: Generate Go structs
test_cases.each do |test|
  # Escape and format incoming fields without quotes
  user = "#{escape_go_string(test[:user])}"
  channel = "#{escape_go_string(test[:channel])}"
  message = "\"#{escape_go_string(test[:message])}\""
  threaded = test[:threaded] ? 'true' : 'false'

  # Format replies
  if test[:replies].empty?
    replies = ""
  else
    replies = test[:replies].join(', ')
  end

  # Format events
  if test[:events].empty?
    events = ""
  else
    events = test[:events].join(', ')
  end

  # Print the Go struct line with a leading tab
  puts "\t{#{user}, #{channel}, #{message}, #{threaded}, []TestMessage{#{replies}}, []Event{#{events}}, #{test[:pause]}},"
end
