package main
import (
	"net/url"
	"strings"
	"time"
	"regexp"
	"strconv"
	"fmt"
)

/* Returns the action */
func processAction (entry *DBEntry, query url.Values) string {
	if len(query.Get("entry")) != 0 && query.Get("entry")[0] == '-' {
		return "Add"
	} else if !entry.Editable() {
		return "View"
	} else {
		return "Edit"
	}
}

func processBlock (in []byte, action string) []byte {
	fields := strings.Fields(string(in))
	if len(fields) < 3 {
		return nil
	}
	not := fields[0] == "ifnot"
	if not != (fields[1] == action) {
		return []byte(strings.Join(fields[2:], " "))
	}
	return nil
}

/* Perform <!--[ ... ]--> substitutions */
func processBlocks (indata []byte, action string) []byte {
	out := string(indata)
	start, end := -1, -1
	count := 0
	hasNested := false
	for index := 0; index < len(out); index++ {
		if strings.HasPrefix(out[index:], "<!--[") {
			if count == 0 {
				start = index			
			} else {
				hasNested = true
			}
			count++
		}
		if strings.HasPrefix(out[index:], "]-->") {
			count--
			if count == 0 {
				end = index
				parsed := processBlock([]byte(out[start+5:end]), action)
				out = string(out[0:start]) + string(parsed) + string(out[end+4:])
				index -= (end+4 - start) - len(parsed)
				start, end = -1, -1
			}
		}
	}

	// [5 [7 2] [5 3] 6]
	if hasNested {
		return processBlocks ([]byte(out), action)
	} else {
		return []byte(out)
	}
}

func process(in []byte, user UserData, entry *DBEntry, entryIndex int) ([]byte, error) {
	action := ""
	if entryIndex < 0 || entry == nil {
		action = ACTION_ADD
	} else if entry.Editable() {
		action = ACTION_EDIT
	} else {
		action = ACTION_VIEW
	}

	in = processBlocks(in, action)
	
	return regexp.MustCompile("(?s)\\[\\[.*?\\]\\]").ReplaceAllFunc(in, func (rawcode []byte) []byte {
		code := string(rawcode[2:len(rawcode)-2])
		cmd := strings.Fields(code)
		if len(cmd) < 1 {
			return nil
		}
		switch cmd[0] {
		case "date":
			if len(cmd) < 2 { return []byte(time.Now().Format("2006-01-02")) }
			diff, err := strconv.Atoi(cmd[1])
			if err != nil {
				return nil
			}
			return []byte(time.Now().AddDate(0,0,diff).Format("2006-01-02"))
		case "user":
			if len(cmd) != 2 { return nil }
			if cmd[1] == "name" {
				return []byte(user.Name)
			}
			if cmd[1] == "email" {
				return []byte(user.Email)
			}
			if cmd[1] == "total" {
				return []byte(fmt.Sprint(DBTotal(user.Email)))
			}
			return nil
		case "entry":
			if len(cmd) != 2 || entry == nil { return nil }
			if cmd[1] == "name" { 
				return []byte(entry.Name)
			}
			if cmd[1] == "index" { 
				return []byte(fmt.Sprint(entryIndex))
			}
			if cmd[1] == "hours" { 
				return []byte(fmt.Sprint(entry.Hours))
			}
			if cmd[1] == "date" { 
				return []byte(entry.Date.Format("2006-01-02"))
			}
			if cmd[1] == "org" { 
				return []byte(entry.Organization)
			}
			if cmd[1] == "contact.name" { 
				return []byte(entry.ContactName)
			}
			if cmd[1] == "contact.email" { 
				return []byte(entry.ContactEmail)
			}
			if cmd[1] == "contact.phone" { 
				if entry.ContactPhone != 0 {
					str := fmt.Sprint(entry.ContactPhone)
					if len(str) == 10 {
						return []byte(str[0:3] + "-" + str[3:6] + "-" + str[6:])
					} else if len(str) == 11 && str[0] == '1' {
						return []byte("+1 " + str[1:4] + "-" + str[4:7] + "-" + str[7:])
					} else {
						return []byte("+" + str)
					}
				}
			}
			if cmd[1] == "action" {
				return []byte(action)
			}
			if cmd[1] == "disabled" {
				if action != ACTION_VIEW {
					return nil
				} else {
					return []byte("disabled")
				}
			}
			return nil
		case "repeat": // TODO: Change to block
			html := strings.Trim(code[6:], " \t\n")
			out := ""
			for i, entry := range DBList(user.Email) {
				if entry != nil {
					out += strings.NewReplacer("[index]", fmt.Sprint(i), "[name]", entry.Name,  "[hours]", strconv.FormatUint(uint64(entry.Hours), 10)).Replace(html)
				}
			}
			return []byte(out)
		default:
			return nil				
		}
	}), nil
}
