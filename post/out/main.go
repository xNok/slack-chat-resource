package main

import (
    "encoding/json"
    "io/ioutil"
    "os"
    "path/filepath"
    "fmt"
    "strings"
    "github.com/jleben/slack-chat-resource/utils"
    "github.com/nlopes/slack"
)

func main() {
	if len(os.Args) < 2 {
		println("usage: " + os.Args[0] + " <source>")
		os.Exit(1)
	}

    source_dir := os.Args[1]

    var request utils.OutRequest

    request_err := json.NewDecoder(os.Stdin).Decode(&request)
    if request_err != nil {
        fatal("Parsing request.", request_err)
    }

    if len(request.Source.Token) == 0 {
        fatal1("Missing source field: token.")
    }

    if len(request.Source.ChannelId) == 0 {
        fatal1("Missing source field: channel_id.")
    }

    if len(request.Params.MessageFile) == 0 && request.Params.Message == nil {
        fatal1("Missing params field: message or message_file.")
    }

    var message *utils.OutMessage

    if len(request.Params.MessageFile) != 0 {
        message = new(utils.OutMessage)
        read_message_file(filepath.Join(source_dir,request.Params.MessageFile), message)
    } else {
        message = request.Params.Message
        interpolate_message(message, source_dir, &request)
    }

    {
        fmt.Fprintf(os.Stderr, "About to send this message:\n")
        m, _ := json.MarshalIndent(message, "", "  ")
        fmt.Fprintf(os.Stderr, "%s\n", m)
    }

    slack_client := slack.New(request.Source.Token)

    response := send(message, &request, slack_client)

    response_err := json.NewEncoder(os.Stdout).Encode(&response)
    if response_err != nil {
        fatal("encoding response", response_err)
    }
}

func read_message_file(path string, message *utils.OutMessage) {
    file, open_err := os.Open(path)
    if open_err != nil {
        fatal("opening message file", open_err)
    }

    read_err := json.NewDecoder(file).Decode(message)
    if read_err != nil {
        fatal("reading message file", read_err)
    }
}

func interpolate_message(message *utils.OutMessage, source_dir string, request *utils.OutRequest) {
    message.Text = interpolate(message.Text, source_dir, request)
    message.ThreadTimestamp = interpolate(message.ThreadTimestamp, source_dir, request)

    for i := 0; i < len(message.Attachments); i++ {
        attachment := &message.Attachments[i]

        attachment.Fallback = interpolate(attachment.Fallback, source_dir, request)
        attachment.Title = interpolate(attachment.Title, source_dir, request)
        attachment.TitleLink = interpolate(attachment.TitleLink, source_dir, request)
        attachment.Pretext = interpolate(attachment.Pretext, source_dir, request)
        attachment.Text = interpolate(attachment.Text, source_dir, request)
        attachment.Footer = interpolate(attachment.Footer, source_dir, request)

        for j := 0; j < len(attachment.Fields); j++ {
            field := &attachment.Fields[j]
            field.Title = interpolate(field.Title, source_dir, request)
            field.Value = interpolate(field.Value, source_dir, request)
        }

        for k := 0; k < len(attachment.Actions); k++ {
            action := &attachment.Actions[k]
            action.Text = interpolate(action.Text, source_dir, request)
            action.URL = interpolate(action.URL, source_dir, request)
        }
    }
}

func get_file_contents(path string) string {
    file, open_err := os.Open(path)
    if open_err != nil {
        fatal("opening file", open_err)
    }

    data, read_err := ioutil.ReadAll(file)
    if read_err != nil {
        fatal("reading file", read_err)
    }

    text := string(data) 

    // clean the string from \n in last possition
    if text[len(text)-2:] == "\n" {
        text = text[:len(text)-2]
    }
        
    return text
}

func interpolate(text string, source_dir string, request *utils.OutRequest) string {

    var out_text string

    start_var := 0
    end_var := 0
    inside_var := false
    c0 := '_'

    for pos, c1 := range text {
        if inside_var {
            if c0 == '}' && c1 == '}' {
                inside_var = false
                end_var = pos + 1

                var value string
                var var_name_proc []string

                if text[start_var+2] == '$' {
                    var_name := text[start_var+3:end_var-2]
                    var_name_proc = strings.Split(var_name, "|")
                    var_name = var_name_proc[0]
                    value = os.Getenv(var_name)
                } else {
                    var_name := text[start_var+2:end_var-2]
                    var_name_proc = strings.Split(var_name, "|")
                    var_name = var_name_proc[0]
                    value = get_file_contents(filepath.Join(source_dir, var_name))
                }

                if len(var_name_proc) > 1{
                    if var_name_proc[1] == "blame" {
                        fmt.Fprintf(os.Stderr, "About to apply blame:\n")
                        fmt.Fprintf(os.Stderr, value)
                        fmt.Fprintf(os.Stderr, "\n")
                        m, _ := json.MarshalIndent(request.Source.SlackUserMap, "", "  ")
                        fmt.Fprintf(os.Stderr, "%s\n", m)
                        fmt.Fprintf(os.Stderr, "\n")
                        value = request.Source.SlackUserMap[value]
                    }
                }

                out_text += value
            }
        } else {
            if c0 == '{' && c1 == '{' {
                inside_var = true
                start_var = pos - 1
                out_text += text[end_var:start_var]
            }
        }
        c0 = c1
    }

    out_text += text[end_var:]

    return out_text
}

func send(message *utils.OutMessage, request *utils.OutRequest, slack_client *slack.Client) utils.OutResponse {

    _, timestamp, err := slack_client.PostMessage(request.Source.ChannelId,
                                                    slack.MsgOptionText(message.Text, false),
                                                    slack.MsgOptionAttachments(message.Attachments...),
                                                    slack.MsgOptionPostMessageParameters(message.PostMessageParameters))

    if err != nil {
        fatal("sending", err)
    }

    var response utils.OutResponse
    response.Version = utils.Version { "timestamp": timestamp }
    return response
}

func fatal(doing string, err error) {
    fmt.Fprintf(os.Stderr, "Error " + doing + ": " + err.Error() + "\n")
    os.Exit(1)
}

func fatal1(reason string) {
    fmt.Fprintf(os.Stderr, reason + "\n")
    os.Exit(1)
}
