// Copyright 2004-2011 DTRules.com, Inc.
// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operators

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	Register("concat", opConcat)
	Register("s+", opConcat)
	Alias("s+", "strconcat")
	Register("substring", opSubstring)
	Register("trim", opTrim)
	Register("strtrim", opTrim)
	Register("lowercase", opLowercase)
	Register("tolowercase", opLowercase)
	Register("uppercase", opUppercase)
	Register("touppercase", opUppercase)
	Register("stringlength", opStringLength)
	Register("strlength", opStringLength)
	Register("indexof", opIndexOf)
	Register("startswith", opStartsWith)
	Register("endswith", opEndsWith)
	Register("contains", opContains)
	Register("replace", opReplace)
	Register("split", opSplit)
	Register("tostring", opToString)
	Register("cvs", opCvs) // Convert to string (PostScript style)
	Register("regexmatch", opRegexMatch)
	Register("s==", opStringEqual)
	Alias("s==", "streq")
	Register("s==i", opStringEqualIgnoreCase)
	Alias("s==i", "streqignorecase")
	Register("s<", opStringLT)
	Register("s>", opStringGT)
	Register("s<=", opStringLE)
	Register("s>=", opStringGE)
}

// opConcat: ( string1 string2 -- string ) concatenates two strings
func opConcat(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}

	result := s1.StringValue() + s2.StringValue()
	return state.DataPush(dtrules.NewRString(result))
}

// opSubstring: ( string start length -- substring )
// Returns substring starting at 'start' with 'length' characters
func opSubstring(state dtrules.State) error {
	lengthObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	str := strObj.StringValue()
	start, err := startObj.IntValue()
	if err != nil {
		return err
	}
	length, err := lengthObj.IntValue()
	if err != nil {
		return err
	}

	// Handle bounds
	runes := []rune(str)
	strLen := len(runes)

	if start < 0 {
		start = 0
	}
	if start > strLen {
		start = strLen
	}
	if length < 0 {
		length = 0
	}
	if start+length > strLen {
		length = strLen - start
	}

	result := string(runes[start : start+length])
	return state.DataPush(dtrules.NewRString(result))
}

// opTrim: ( string -- string ) trims whitespace from both ends
func opTrim(state dtrules.State) error {
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.TrimSpace(strObj.StringValue())
	return state.DataPush(dtrules.NewRString(result))
}

// opLowercase: ( string -- string ) converts to lowercase
func opLowercase(state dtrules.State) error {
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.ToLower(strObj.StringValue())
	return state.DataPush(dtrules.NewRString(result))
}

// opUppercase: ( string -- string ) converts to uppercase
func opUppercase(state dtrules.State) error {
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.ToUpper(strObj.StringValue())
	return state.DataPush(dtrules.NewRString(result))
}

// opStringLength: ( string -- length ) returns string length
func opStringLength(state dtrules.State) error {
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	// Use rune count for Unicode support
	length := len([]rune(strObj.StringValue()))
	return state.DataPush(dtrules.GetRIntegerValueFromInt(length))
}

// opIndexOf: ( string substring -- index ) returns index of substring, or -1
func opIndexOf(state dtrules.State) error {
	subObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	index := strings.Index(strObj.StringValue(), subObj.StringValue())
	return state.DataPush(dtrules.GetRIntegerValueFromInt(index))
}

// opStartsWith: ( string prefix -- boolean )
func opStartsWith(state dtrules.State) error {
	prefixObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.HasPrefix(strObj.StringValue(), prefixObj.StringValue())
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opEndsWith: ( string suffix -- boolean )
func opEndsWith(state dtrules.State) error {
	suffixObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.HasSuffix(strObj.StringValue(), suffixObj.StringValue())
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opContains: ( string substring -- boolean )
func opContains(state dtrules.State) error {
	subObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.Contains(strObj.StringValue(), subObj.StringValue())
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opReplace: ( string old new -- string ) replaces all occurrences
func opReplace(state dtrules.State) error {
	newObj, err := state.DataPop()
	if err != nil {
		return err
	}
	oldObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	result := strings.ReplaceAll(strObj.StringValue(), oldObj.StringValue(), newObj.StringValue())
	return state.DataPush(dtrules.NewRString(result))
}

// opSplit: ( string delimiter -- array ) splits string by delimiter
func opSplit(state dtrules.State) error {
	delimObj, err := state.DataPop()
	if err != nil {
		return err
	}
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	parts := strings.Split(strObj.StringValue(), delimObj.StringValue())
	elements := make([]dtrules.Object, len(parts))
	for i, part := range parts {
		elements[i] = dtrules.NewRString(part)
	}

	arr, err := dtrules.NewArrayWithElements(state.GetSession(), false, elements, false)
	if err != nil {
		return err
	}
	return state.DataPush(arr)
}

// opToString: ( obj -- string ) converts any object to string
func opToString(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}

	return state.DataPush(dtrules.NewRString(obj.StringValue()))
}

// opCvs: ( obj -- string ) converts to string (PostScript style)
func opCvs(state dtrules.State) error {
	return opToString(state)
}

// opCapitalize capitalizes the first letter of a string
func opCapitalize(state dtrules.State) error {
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}

	str := strObj.StringValue()
	if len(str) == 0 {
		return state.DataPush(dtrules.NewRString(""))
	}

	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return state.DataPush(dtrules.NewRString(string(runes)))
}

// opRegexMatch: ( regex string -- boolean ) returns true if string matches regex
func opRegexMatch(state dtrules.State) error {
	strObj, err := state.DataPop()
	if err != nil {
		return err
	}
	regexObj, err := state.DataPop()
	if err != nil {
		return err
	}

	str := strObj.StringValue()
	pattern := regexObj.StringValue()

	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		return state.DataPush(dtrules.GetRBoolean(false))
	}
	return state.DataPush(dtrules.GetRBoolean(matched))
}

// opStringEqual: ( string1 string2 -- boolean ) returns true if strings are equal
func opStringEqual(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(s1.StringValue() == s2.StringValue()))
}

// opStringEqualIgnoreCase: ( string1 string2 -- boolean ) case-insensitive compare
func opStringEqualIgnoreCase(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(strings.EqualFold(s1.StringValue(), s2.StringValue())))
}

// opStringLT: ( string1 string2 -- boolean ) returns true if string1 < string2
func opStringLT(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(s1.StringValue() < s2.StringValue()))
}

// opStringGT: ( string1 string2 -- boolean ) returns true if string1 > string2
func opStringGT(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(s1.StringValue() > s2.StringValue()))
}

// opStringLE: ( string1 string2 -- boolean ) returns true if string1 <= string2
func opStringLE(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(s1.StringValue() <= s2.StringValue()))
}

// opStringGE: ( string1 string2 -- boolean ) returns true if string1 >= string2
func opStringGE(state dtrules.State) error {
	s2, err := state.DataPop()
	if err != nil {
		return err
	}
	s1, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(s1.StringValue() >= s2.StringValue()))
}
