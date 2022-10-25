// SPDX-License-Identifier: BSD-3-Clause

package completer

import (
	"strings"

	"github.com/knz/bubbline/computil"
	"github.com/knz/bubbline/editline"
)

func Autocomplete(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	// Detect the word under the cursor.
	word, wstart, wend := computil.FindWord(v, line, col)

	// Is this a part of the word "hello"?
	const specialWord = "HELLO"
	if !strings.HasPrefix(specialWord, strings.ToUpper(word)) {
		return msg, nil
	}

	return msg, editline.SingleWordCompletion(specialWord, col, wstart, wend)
}
