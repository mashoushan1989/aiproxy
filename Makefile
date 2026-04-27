.PHONY: check check-quick check-backend branch

check:
	bash scripts/check.sh

check-quick:
	bash scripts/check.sh --quick

check-backend:
	bash scripts/check.sh --backend-only

branch:
	@test -n "$(NAME)" || (echo "Usage: make branch NAME=fix/short-name" >&2; exit 1)
	bash scripts/new-task-branch.sh "$(NAME)"
