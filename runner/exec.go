package runner

import (
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus"
)

type execResult string

const (
	execResultBlockError            execResult = "block_error"
	execResultInBlock               execResult = "in_block"
	execResultCanNotBlock           execResult = "can_not_block"
	execResultExecError             execResult = "exec_error"
	execResultExecErrorWithoutBlock execResult = "exec_error_without_block"
	execResultSuccess               execResult = "success"
	execResultSuccessWithoutBlock   execResult = "success_without_block"
)

func (r execResult) String() string {
	return string(r)
}

func exec(task executor.Task, blocker blocker, logger *logrus.Logger) (execResult, error) {
	if task.BlockTTL().Seconds() <= 0 {
		err := task.Exec(logger)
		if err != nil {
			return execResultExecErrorWithoutBlock, err
		}

		return execResultSuccessWithoutBlock, nil
	}

	success, err := blocker.BlockInProgress(task.Fingerprint())
	if err != nil {
		return execResultBlockError, err
	}

	if !success {
		return execResultInBlock, nil
	}

	err = task.Exec(logger)
	if err != nil {
		blocker.Unblock(task.Fingerprint())
		return execResultExecError, err
	}

	err = blocker.BlockForTTL(task.Fingerprint(), task.BlockTTL())
	if err != nil {
		return execResultCanNotBlock, err
	}

	return execResultSuccess, nil
}
