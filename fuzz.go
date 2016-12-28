// +build gofuzz

package docopt

/* -- Fuzz tests usage

$ go get github.com/dvyukov/go-fuzz/go-fuzz
$ go get github.com/dvyukov/go-fuzz/go-fuzz-build

$ go-fuzz-build github.com/aviddiviner/docopt-go
$ go-fuzz -bin docopt-fuzz.zip -workdir=workdir

-- Building the workdir/corpus from testcases.docopt (Ruby code)

FileUtils.rm_r('workdir')
FileUtils.mkdir_p('workdir/corpus')
File.read("testcases.docopt").split(/r"""/)[1..-1].each_with_index do |chunk, t|
  testcase = chunk.gsub(/^#.*$/, '').strip + "\n"
  doc, cases = testcase.split(/"""/)
  cases.split(/\$/)[1..-1].map(&:lines).each_with_index do |body, i|
    output = [doc, body.shift, body.join].map(&:strip).join("\n~~~\n")
    File.write("workdir/corpus/#{"testcase-%02d-%01d" % [t, i]}", output)
  end
end

*/

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func Fuzz(data []byte) int {
	tc, err := parseFuzzyTestcase(data)
	if err != nil {
		return 0
	}
	// var invalidUsage bool
	// var testParser = &Parser{HelpHandler: func(err error, usage string) { invalidUsage = (err != nil) }}

	var testParser = &Parser{HelpHandler: NoHelpHandler}
	result, err := testParser.ParseArgs(tc.doc, tc.argv, "")

	if _, ok := err.(*UserError); ok && tc.userError {
		// testcase says we should get a user-error, and we got one
		return 1
	}
	if reflect.DeepEqual(tc.expect, result) == true {
		// we got the result our testcase predicted
		return 1
	}

	return 0
}

type fuzztest struct {
	doc       string
	prog      string
	argv      []string
	expect    Opts
	userError bool
}

var errParseFuzz = fmt.Errorf("error parsing fuzzy testcase")

func parseFuzzyTestcase(raw []byte) (*fuzztest, error) {
	split := strings.SplitN(string(raw), "\n~~~\n", 3)
	if len(split) < 3 {
		return nil, errParseFuzz
	}

	doc := split[0]
	prog, _, argvString := stringPartition(strings.TrimSpace(split[1]), " ")
	expectString := split[2]

	var argv []string
	if len(argvString) > 0 {
		argv = strings.Fields(argvString)
	}

	var expectUntyped interface{}
	if err := json.Unmarshal([]byte(expectString), &expectUntyped); err != nil {
		return nil, err
	}

	switch expect := expectUntyped.(type) {
	case string:
		if expect == "user-error" {
			return &fuzztest{doc, prog, argv, nil, true}, nil
		}
	case map[string]interface{}:
		// convert []interface{} values to []string
		// convert float64 values to int
		for k, vUntyped := range expect {
			switch v := vUntyped.(type) {
			case []interface{}:
				itemList := make([]string, len(v))
				for i, itemUntyped := range v {
					if item, ok := itemUntyped.(string); ok {
						itemList[i] = item
					} else {
						return nil, errParseFuzz
					}
				}
				expect[k] = itemList
			case float64:
				expect[k] = int(v)
			default:
				return nil, errParseFuzz
			}
		}
		return &fuzztest{doc, prog, argv, expect, false}, nil
	}

	return nil, errParseFuzz
}
