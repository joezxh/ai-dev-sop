package llm

import (
	"context"
	"fmt"
)

type Fake struct {
	answer string
	n      int
}

func NewFake(answer string) *Fake { return &Fake{answer: answer} }

func (f *Fake) Complete(ctx context.Context, req Request) (*Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in := ""
	if len(req.Messages) > 0 {
		in = req.Messages[len(req.Messages)-1].Content
	}
	return &Response{
		Text:   fmt.Sprintf("%s /* inputs=%q system=%q */", f.answer, in, req.System),
		Tokens: len(in),
	}, nil
}

func (f *Fake) Name() string { return "fake" }
