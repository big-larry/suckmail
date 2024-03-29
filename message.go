package suckmail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"strings"
	textTemplate "text/template"

	"github.com/antchfx/htmlquery"
	"github.com/big-larry/suckutils"
	"golang.org/x/net/html"
)

type MailMessage struct {
	fromEmail       string // Почта отправителя
	fromName        string // Имя отправителя
	subject         string // Тема письма
	html            string // Тело письма в HTML
	plain           string // Тело письма простым текстом
	recieverEmail   string // Почта получателя
	recieverName    string // Имя получателя
	recieverCCEmail string // Почта получателя
	recieverCCName  string // Имя получателя
	replyTo         string // Кому отвечать
	replyToName     string // Кому отвечать
	unsubscribe     string // Заголовок "List-Unsubscribe" TODO
	htmlLink        string // Ссылка на письмо в браузере TODO
	attachment      []MailAttachment
	errors          []error
	Data            interface{}
}

type MailAttachment struct {
	id          string
	name        string
	contentType string
	data        []byte
}

type mailMessageError struct {
	method string
	error  error
}

func newMailMessageErrorFromError(method string, err error) *mailMessageError {
	return &mailMessageError{method, err}
}
func newMailMessageErrorFromString(method, err string) *mailMessageError {
	return &mailMessageError{method, errors.New(err)}
}

func (err *mailMessageError) Error() string {
	return suckutils.ConcatThree(err.method, ": ", err.error.Error())
}

func NewMessage() *MailMessage {
	return &MailMessage{errors: make([]error, 0)}
}

func (msg *MailMessage) HasErrors() bool {
	return len(msg.errors) > 0
}

func (msg *MailMessage) GetErrors() []error {
	return msg.errors
}

func (msg *MailMessage) SetHTMLFromTemplate(t *htmlTemplate.Template, generatePlainText bool) *MailMessage {
	if t == nil {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetHTMLFromTemplate", "Template is empty"))
		return msg
	}
	var w strings.Builder
	err := t.Execute(&w, msg)
	if err != nil {
		msg.errors = append(msg.errors, newMailMessageErrorFromError("SetHTMLFromTemplate", err))
		return msg
	}
	return msg.SetHTML(w.String(), generatePlainText)
}

func (msg *MailMessage) SetData(data interface{}) *MailMessage {
	msg.Data = data
	return msg
}

func (msg *MailMessage) SetPlainTextFromTemplate(t *textTemplate.Template) *MailMessage {
	if t == nil {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetPlainTextFromTemplate", "Template is empty"))
		return msg
	}
	var w strings.Builder
	err := t.Execute(&w, msg)
	if err != nil {
		msg.errors = append(msg.errors, newMailMessageErrorFromError("SetPlainTextFromTemplate", err))
		return msg
	}
	return msg.SetPlainText(w.String())
}

func (msg *MailMessage) SetPlainText(text string) *MailMessage {
	if text == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetPlainText", "Empty string"))
		return msg
	}
	msg.plain = text
	return msg
}

func (msg *MailMessage) SetHTML(body string, generatePlainText bool) *MailMessage {
	if body == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetHTML", "Empty HTML"))
		return msg
	}
	if generatePlainText {
		txt, err := generatePlainTextFromHTML(body)
		if err != nil {
			msg.errors = append(msg.errors, newMailMessageErrorFromString("SetHTML - GeneratePlainTextFromHTML", "Empty HTML"))
			return msg
		}
		msg.SetPlainText(txt)
	}
	msg.html = body
	return msg
}

func (msg *MailMessage) GetPlainText() string {
	return msg.plain
}

func (msg *MailMessage) GetHTMLLen() int {
	return len(msg.html)
}

func (msg *MailMessage) SetFrom(email, name, reply string) *MailMessage {
	if email == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetFrom", "Empty email"))
		return msg
	}
	if name == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetFrom", "Empty name"))
		return msg
	}
	msg.fromEmail = email
	msg.fromName = name
	msg.replyTo = reply
	return msg
}

func (msg *MailMessage) SetReplyTo(email, name string) *MailMessage {
	if email == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetReplyTo", "Empty email"))
		return msg
	}
	msg.replyTo = email
	msg.replyToName = name
	return msg
}

func (msg *MailMessage) SetReciever(email, name string) *MailMessage {
	if email == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetReciever", "Empty email"))
		return msg
	}
	// if name == "" {
	// 	msg.errors = append(msg.errors, newMailMessageErrorFromString("SetReciever", "Empty name"))
	// 	return msg
	// }
	msg.recieverEmail = email
	msg.recieverName = name
	return msg
}

func (msg *MailMessage) SetRecieverCC(email, name string) *MailMessage {
	if email == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetRecieverCC", "Empty email"))
		return msg
	}
	msg.recieverCCEmail = email
	msg.recieverCCName = name
	return msg
}

func (msg *MailMessage) SetSubject(subject string) *MailMessage {
	if subject == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetSubject", "Empty string"))
		return msg
	}
	msg.subject = subject
	return msg
}

func (msg *MailMessage) SetHTMLLink(link string) *MailMessage {
	if link == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetHTMLLink", "Empty string"))
		return msg
	}
	msg.htmlLink = link
	return msg
}

func (msg *MailMessage) SetUnsubscribeLink(link string) *MailMessage {
	if link == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetUnsubscribe", "Empty string"))
		return msg
	}
	if len(msg.unsubscribe) > 0 {
		msg.unsubscribe = strings.Join([]string{msg.unsubscribe, suckutils.Concat("<", link, ">")}, ",")
	}
	return msg
}
func (msg *MailMessage) SetUnsubscribeMail(email string) *MailMessage {
	if email == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("SetUnsubscribe", "Empty string"))
		return msg
	}
	if len(msg.unsubscribe) > 0 {
		msg.unsubscribe = strings.Join([]string{msg.unsubscribe, suckutils.Concat("<mailto:", email, "?subject=unsubscribe>")}, ",")
	}
	return msg
}

func (msg *MailMessage) AddAttachment(attachment *MailAttachment) *MailMessage {
	if attachment.name == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("AddAttachment", "Empty name"))
		return msg
	}
	if attachment.contentType == "" {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("AddAttachment", "Empty contentType"))
		return msg
	}
	if len(attachment.data) == 0 {
		msg.errors = append(msg.errors, newMailMessageErrorFromString("AddAttachment", "Empty data"))
		return msg
	}
	if msg.attachment == nil {
		msg.attachment = make([]MailAttachment, 0)
	}
	msg.attachment = append(msg.attachment, *attachment)
	return msg
}

func NewMailAttachment(id, name, contentType string, data []byte) *MailAttachment {
	return &MailAttachment{id: id, name: name, contentType: contentType, data: data}
}

func (a *MailAttachment) GetName() string {
	return a.name
}
func (a *MailAttachment) GetId() string {
	return a.id
}
func (a *MailAttachment) GetLen() int {
	return len(a.data)
}

func (msg *MailMessage) GetRecieverName() string {
	return msg.recieverName
}

func (msg *MailMessage) Build() ([]byte, error) {
	buf := &bytes.Buffer{}
	if msg.fromEmail != "" {
		if msg.fromName != "" {
			fmt.Fprintf(buf, "From: =?UTF-8?B?%s?= <%s>\r\n", base64.StdEncoding.EncodeToString([]byte(msg.fromName)), msg.fromEmail)
		} else {
			fmt.Fprintf(buf, "From: %s\r\n", msg.fromEmail)
		}
	}
	fmt.Fprintf(buf, "To: =?UTF-8?B?%s?= <%s>\r\n", base64.StdEncoding.EncodeToString([]byte(msg.recieverName)), msg.recieverEmail)
	if msg.recieverCCEmail != "" {
		fmt.Fprintf(buf, "Cc: =?UTF-8?B?%s?= <%s>\r\n", base64.StdEncoding.EncodeToString([]byte(msg.recieverCCName)), msg.recieverCCEmail)
	}
	if msg.replyTo != "" {
		fmt.Fprintf(buf, "Reply-To: =?UTF-8?B?%s?= <%s>\r\n", msg.replyToName, msg.replyTo)
	} else {
		fmt.Fprintf(buf, "Reply-To: =?UTF-8?B?%s?= <%s>\r\n", msg.fromName, msg.fromEmail)
	}
	fmt.Fprintf(buf, "Subject: =?UTF-8?B?%s?=\r\n", base64.StdEncoding.EncodeToString([]byte(msg.subject)))
	if msg.unsubscribe != "" {
		fmt.Fprintf(buf, "List-Unsubscribe: %s\r\n", msg.unsubscribe)
		// fmt.Fprintln(buf, "List-Unsubscribe-Post: List-Unsubscribe=One-Click")
	}
	if len(msg.attachment) == 0 {
		fmt.Fprintln(buf, "Content-Type: multipart/alternative; boundary=\"===============main==\"")
		fmt.Fprintln(buf, "MIME-Version: 1.0")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "This is a message in Mime Format. If you see this, your mail reader does not support this format.")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "--===============main==")
		fmt.Fprintln(buf, "Content-Type: text/plain; charset=utf-8")
		fmt.Fprintln(buf, "Content-Transfer-Encoding: base64")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, base64.StdEncoding.EncodeToString([]byte(msg.plain)))
		fmt.Fprintln(buf, "--===============main==")
		fmt.Fprintln(buf, "Content-Type: text/html; charset=utf-8")
		fmt.Fprintln(buf, "Content-Transfer-Encoding: base64")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, base64.StdEncoding.EncodeToString([]byte(msg.html)))
	} else {
		images_count := 0
		files_count := 0
		for _, img := range msg.attachment {
			if img.id == "" {
				files_count++
			} else {
				images_count++
			}
		}

		if files_count > 0 {
			fmt.Fprintln(buf, "Content-Type: multipart/mixed; boundary=\"===============main==\"")
		} else {
			fmt.Fprintln(buf, "Content-Type: multipart/alternative; boundary=\"===============main==\"")
		}
		fmt.Fprintln(buf, "MIME-Version: 1.0")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "This is a message in Mime Format. If you see this, your mail reader does not support this format.")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "--===============main==")
		fmt.Fprintln(buf, "Content-Type: multipart/alternative;boundary=\"===============boundary==\"")
		fmt.Fprintln(buf)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "--===============boundary==")
		fmt.Fprintln(buf, "Content-Type: text/plain; charset=utf-8")
		fmt.Fprintln(buf, "Content-Transfer-Encoding: base64")
		fmt.Fprintln(buf)
		printBase64(buf, []byte(msg.plain))
		fmt.Fprintln(buf, "--===============boundary==")
		if images_count > 0 {
			fmt.Fprintln(buf, "Content-Type: multipart/related;boundary=\"===============boundary2==\"")
			fmt.Fprintln(buf)
			fmt.Fprintln(buf)
			fmt.Fprintln(buf, "--===============boundary2==")
		}
		fmt.Fprintln(buf, "Content-Type: text/html; charset=utf-8")
		fmt.Fprintln(buf, "Content-Transfer-Encoding: base64")
		fmt.Fprintln(buf)
		printBase64(buf, []byte(msg.html))

		if images_count > 0 {
			for _, img := range msg.attachment {
				if img.id == "" {
					continue
				}
				fmt.Fprintln(buf, "--===============boundary2==")
				fmt.Fprintf(buf, "Content-Type: %s; name=\"=?UTF-8?B?%s?=\"\r\n", img.contentType, base64.StdEncoding.EncodeToString([]byte(img.name)))
				fmt.Fprintf(buf, "Content-Disposition: inline; filename=\"=?UTF-8?B?%s?=\"\r\n", base64.StdEncoding.EncodeToString([]byte(img.name)))
				fmt.Fprintf(buf, "Content-ID: <%s>\r\n", img.id)
				fmt.Fprintln(buf, "Content-Transfer-Encoding: base64")
				fmt.Fprintln(buf)
				printBase64(buf, img.data)
			}
			fmt.Fprintln(buf, "--===============boundary2==--")
		}
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "--===============boundary==--")
		fmt.Fprintln(buf)
		if files_count > 0 {
			for _, img := range msg.attachment {
				if img.id != "" {
					continue
				}
				fmt.Fprintln(buf, "--===============main==")
				fmt.Fprintf(buf, "Content-Type: %s; name=\"=?UTF-8?B?%s?=\"\r\n", img.contentType, base64.StdEncoding.EncodeToString([]byte(img.name)))
				fmt.Fprintf(buf, "Content-Disposition: attachment; filename=\"=?UTF-8?B?%s?=\"\r\n", base64.StdEncoding.EncodeToString([]byte(img.name)))
				fmt.Fprintln(buf, "Content-Transfer-Encoding: base64")
				fmt.Fprintln(buf)
				printBase64(buf, img.data)
			}
		}
	}
	fmt.Fprintln(buf, "--===============main==--")
	return buf.Bytes(), nil
}

func printBase64(buf io.Writer, data []byte) {
	s := base64.StdEncoding.EncodeToString(data)
	for i := 0; i < len(s); i += 76 {
		if i < len(s)-76 && len(s) > 76 {
			fmt.Fprint(buf, s[i:i+76])
		} else {
			fmt.Fprint(buf, s[i:])
		}
		fmt.Fprintln(buf)
	}
}

func generatePlainTextFromHTML(body string) (string, error) {
	body = strings.ReplaceAll(body, "</p>", "\r\n</p>")
	body = strings.ReplaceAll(body, "</div>", "\r\n</div>")
	body = strings.ReplaceAll(body, "</li>", "\r\n</li>")
	body = strings.ReplaceAll(body, "</P>", "\r\n</P>")
	body = strings.ReplaceAll(body, "</DIV>", "\r\n</DIV>")
	body = strings.ReplaceAll(body, "</LI>", "\r\n</LI>")
	doc, err := htmlquery.Parse(strings.NewReader(body))
	if err != nil {
		return "", err
	}
	ancors := htmlquery.Find(doc, "//A")
	for _, a := range ancors {
		href := htmlquery.SelectAttr(a, "href")
		text := htmlquery.InnerText(a)
		if href == text {
			continue
		}
		title := htmlquery.SelectAttr(a, "title")
		if text == "" && title == "" {
			img := htmlquery.FindOne(a, "//img")
			if img != nil {
				alt := htmlquery.SelectAttr(a, "alt")
				if alt != "" {
					a.AppendChild(&html.Node{Type: html.TextNode, Data: alt + ": " + href})
					continue
				}
			}
		}
		if text == "" && title != "" {
			a.AppendChild(&html.Node{Type: html.TextNode, Data: title + ": " + href})
			continue
		}
		if text == "" {
			a.AppendChild(&html.Node{Type: html.TextNode, Data: href})
		}
	}

	return htmlquery.InnerText(doc), nil
}
