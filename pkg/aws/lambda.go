package aws

import "os"

func InLambda() bool {
	if lambdaTaskRoot := os.Getenv("LAMBDA_TASK_ROOT"); lambdaTaskRoot != "" {
		return true
	}
	return false
}

func LambdaTmpDir() string {
	if InLambda() {
		return "/tmp/"
	}
	return ""
}
