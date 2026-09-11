package service

import "errors"

// ErrWordNotRecognized is returned by AIProvider.ProcessWord when the given
// input is not a valid English word or short expression suitable for a
// vocabulary card (e.g. it's not English, a greeting, a full sentence, or
// gibberish). WordService surfaces this to the user instead of persisting
// anything.
var ErrWordNotRecognized = errors.New("word not recognized as valid English vocabulary")

// ErrIncompleteWordData is returned by WordService when the AI response was
// parsed successfully but is missing required fields. Per
// docs/vocabulario_bot_documentacao.md (secao 8.2): "nunca persistir uma
// palavra com dados incompletos" — this must never be persisted.
var ErrIncompleteWordData = errors.New("AI response is missing required fields")
