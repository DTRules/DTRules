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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// TestAllOperatorsRegistered pins every operator the engine ships so
// that an accidental drop in any init() fires a loud failure at test
// time. This catches the class of bugs that shipped #694 (cvd
// misregistered) and the pre-#684 `neg` / `round` unregistered-op
// bugs — where the compiler emitted names the operator registry
// didn't know.
//
// To add a new operator: register it via Register() in the
// appropriate file, then add the name here. To rename one: update
// both sites. The list below is kept sorted and matches the output
// of:
//
//     grep -hE 'Register\("[^"]+"' pkg/dtrules/operators/*.go | \
//       grep -oE '"[^"]+"' | sort -u
//
// When this test fails on a rename / refactor, that's the signal to
// decide: is the name change intentional? If so, update this list.
// If not, the commit under review is dropping a public op.
func TestAllOperatorsRegistered(t *testing.T) {
	names := []string{
	"true",
	"false",
	"!=",
	"*",
	"+",
	"-",
	"/",
	"<",
	"<=",
	"==",
	">",
	">=",
	"abort",
	"abs",
	"addarray",
	"addat",
	"adddays",
	"addmonths",
	"add_no_dups",
	"addto",
	"addyears",
	"allocate",
	"and",
	"arraytomark",
	"b!=",
	"b*",
	"b+",
	"b-",
	"b/",
	"b<",
	"b<=",
	"b==",
	"b>",
	"b>=",
	"b58check",
	"babs",
	"bech32",
	"beq",
	"bigintbytes",
	"bmod",
	"bnegate",
	"bytes!=",
	"bytes+",
	"bytes==",
	"bytesbigint",
	"bytesidx",
	"byteslen",
	"bytesslice",
	"ceiling",
	"clear",
	"cleararray",
	"cleartomark",
	"clone",
	"concat",
	"contains",
	"copy",
	"copyelements",
	"counttomark",
	"currentdateinzone",
	"cvb",
	"cvb58check",
	"cvbech32",
	"cvbi",
	"cvbytes",
	"cvd",
	"cvdate",
	"cve",
	"cvfp",
	"cvhex",
	"cvi",
	"cvn",
	"cvs",
	"d+",
	"d-",
	"d<",
	"d==",
	"d>",
	"dateformat",
	"dateformatinzone",
	"dateinzone",
	"days",
	"daysbetween",
	"deallocate",
	"debug",
	"deepcopy",
	"def",
	"doloop",
	"dup",
	"elstmterror",
	"elstmtwarn",
	"endofmonth",
	"endofmonthinzone",
	"endofquarter",
	"endofquarterinzone",
	"endofweek",
	"endofweekinzone",
	"endofweekstarting",
	"endofweekstartinginzone",
	"endofyear",
	"endofyearinzone",
	"endswith",
	"entityfetch",
	"entityid",
	"entityname",
	"entitypop",
	"entitypush",
	"error",
	"execute",
	"executetable",
	"f*",
	"f+",
	"f-",
	"fabs",
	"fdiv",
	"find",
	"findcreateentity",
	"first",
	"firstofmonth",
	"firstofmonthinzone",
	"firstofquarter",
	"firstofquarterinzone",
	"firstofweek",
	"firstofweekinzone",
	"firstofweekstarting",
	"firstofweekstartinginzone",
	"firstofyear",
	"firstofyearinzone",
	"firstpass",
	"floor",
	"fmax",
	"fmin",
	"fnegate",
	"for",
	"forall",
	"forallr",
	"forfirst",
	"forfirstelse",
	"forr",
	"fp!=",
	"fp*",
	"fp+",
	"fp-",
	"fp/",
	"fp<",
	"fp<=",
	"fp==",
	"fp>",
	"fp>=",
	"fpabs",
	"fphalfup/",
	"fpmax",
	"fpmin",
	"fpnegate",
	"fptrunc",
	"get",
	"getat",
	"getdate",
	"getday",
	"getdayofmonth",
	"getdayofmonthinzone",
	"getdaysinmonth",
	"getdaysinmonthinzone",
	"getdayofweek",
	"getdayofweekinzone",
	"getdaysinyear",
	"getdaysinyearinzone",
	"gethour",
	"gethourinzone",
	"getminute",
	"getminuteinzone",
	"getmonth",
	"getsecond",
	"getsecondinzone",
	"gettimestamp",
	"getweekofyear",
	"getweekofyearinzone",
	"getyear",
	"getyearinzone",
	"hex",
	"i",
	"if",
	"ifelse",
	"ignore",
	"InContext",
	"indexof",
	"intersection",
	"intersects",
	"isdefined",
	"isnull",
	"j",
	"k",
	"keccak256",
	"last",
	"length",
	"local!",
	"local@",
	"log_warning",
	"lookup",
	"lowercase",
	"mark",
	"max",
	"memberof",
	"merge",
	"min",
	"mod",
	"months",
	"monthsbetween",
	"negate",
	"newarray",
	"newdate",
	"newdateinzone",
	"newdateinzonewithdst",
	"newentity",
	"not",
	"notnull",
	"now",
	"null",
	"or",
	"over",
	"performaliased",
	"performcatcherror",
	"performtable",
	"pick",
	"pop",
	"print",
	"printtos",
	"pstack",
	">r",
	"r>",
	"randomize",
	"regexmatch",
	"remove",
	"removeat",
	"replace",
	"req",
	"ripemd160",
	"roll",
	"rot",
	"roundto",
	"samecalendardayinzone",
	"samecalendarmonthinzone",
	"samecalendarquarterinzone",
	"samecalendarweekinzone",
	"samecalendarweekstartinginzone",
	"samecalendaryearinzone",
	"s+",
	"s<",
	"s<=",
	"s==",
	"s>",
	"s>=",
	"setdebug",
	"sha256",
	"sha3",
	"s==i",
	"sortarray",
	"sortentities",
	"split",
	"startswith",
	"stringlength",
	"strlength",
	"strtrim",
	"substring",
	"swap",
	"throwexception",
	"today",
	"todayinzone",
	"tolowercase",
	"tostring",
	"touppercase",
	"traceoff",
	"traceon",
	"trim",
	"uppercase",
	"while",
	"xdef",
	"xor",
	"years",
	"yearsbetween",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			rn := dtrules.GetRName(name)
			if rn == nil {
				t.Fatalf("GetRName(%q) returned nil — name is not a valid identifier", name)
			}
			op, ok := Get(rn)
			if !ok {
				t.Fatalf("operator %q is not registered", name)
			}
			if op == nil {
				t.Fatalf("operator %q resolved to nil", name)
			}
		})
	}

	// Cardinality guard: if someone registers a new op without adding
	// it to the list above, the count mismatches and we get a loud
	// signal. Uses the public GetOperatorTable() which returns every
	// registered operator (nil placeholders are filtered).
	registered := 0
	for _, obj := range GetOperatorTable() {
		if obj != nil {
			registered++
		}
	}
	if registered != len(names) {
		t.Errorf("registered-op count drift: registry has %d non-nil "+
			"entries, this test enumerates %d — add the new op(s) to "+
			"the names list (or remove a dropped one)",
			registered, len(names))
	}
}
