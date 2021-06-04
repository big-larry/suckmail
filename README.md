# suckmail

```go
	img, err := ioutil.ReadFile("image.jpg")
	if err != nil {
		println(err.Error())
		return
	}
	doc, err := ioutil.ReadFile("doc.docx")
	if err != nil {
		println(err.Error())
		return
	}
	message := suckmail.NewMessage().
		SetFrom("frommail@mail.ru", "Name", "").
		SetReciever("tomail@mail.ru", "Name").
		SetSubject("Hello").
		SetHTML("Hello!<img src=\"cid:myimage\" alt=\"Title\" />", false).
		SetPlainText("Hello, my friend!").
		AddAttachment("myimage", "image.jpg", "image/jpeg", img).
		AddAttachment("", "document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", doc)

	conn, err := tls.Dial("tcp", "94.100.180.160:465", &tls.Config{ServerName: "smtp.mail.ru"})
	if err != nil {
		println(err.Error())
		return
	}
	defer conn.Close()
	if err = suckmail.Send(conn, "smtp.mail.ru", "frommail@mail.ru", "pass", message); err != nil {
		println(err.Error())
		return
	}
```
