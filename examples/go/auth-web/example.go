package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"gopkg.in/ldap.v2"
)


const (
	host    = "ldap.example.com"
	port    = 389
	userDn  = "uid=<user>,ou=people,dc=example,dc=com"
	topDn 	= "dc=example,dc=com"
)

var (
	fieldsRegExp = regexp.MustCompile(`[^\s,]+`)
)

func search(filter string, fields string, baseDn string) (output string, err error) {
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	defer l.Close()

	search := ldap.NewSearchRequest(
		baseDn,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		fieldsRegExp.FindAllString(fields, -1),
		nil)

	result, err := l.Search(search)
	if err != nil {
		return
	}

	ldapattrs := ""
	for _,entry := range result.Entries {
		ldapattrs += "DN:" + entry.DN + "\n"
		for _,attr := range entry.Attributes {
			ldapattrs += attr.Name + " = " + strings.Join(attr.Values, ",") + "\n"
		}
		ldapattrs += "\n"
	}
	output = ldapattrs

	return
}

func auth(username string, password string) (err error){
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatalf("Failed to connect %v", err)
	}
	defer l.Close()

	l.Debug = true

	// authenticate a user. the object has to have a `userPassword` property.
	user := strings.Replace(userDn, "<user>", username, -1)
	err = l.Bind(user, password)

	return
}

func formatForWeb(input string) (output []byte){
	text := "<!DOCTYPE html><br/><html><br/>" + strings.Replace(input, "\n", "<br/>", -1) + "<br/></html>"
  output = []byte(text)
	return
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		qs := r.URL.Query()
		user := qs.Get("user")
		password := qs.Get("password")

		if len(user) > 0 {
			err := auth(user, password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		filter := "(uid=" + user + ")"
		fields := "*"
		baseDn := topDn

		h, err := search(filter, fields, baseDn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Write(formatForWeb(h))
	})

	fmt.Printf("Listening at http://%s:12345\n", host)

	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatalf("Failed to ListenAndServe: %v", err)
	}
}
