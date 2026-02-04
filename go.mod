module github.com/mtreilly/arc-ai

go 1.21

require (
	github.com/spf13/cobra v1.8.0
	github.com/mtreilly/arc-sdk v0.0.1
	github.com/mtreilly/arc-tmux v0.0.1
)

replace (
	github.com/mtreilly/arc-sdk => ../arc-sdk
	github.com/mtreilly/arc-tmux => ../arc-tmux
)