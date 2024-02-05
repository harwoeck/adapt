package adapt

import "errors"

var ErrIntegrityProtection = errors.New("adapt: abort due to integrity protection rules. See log output for details")
var ErrInvalidSource = errors.New("adapt: source violated a precondition. See log output for details")
