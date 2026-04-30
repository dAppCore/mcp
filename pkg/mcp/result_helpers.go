// SPDX-License-Identifier: EUPL-1.2

package mcp

import core "dappco.re/go"

func resultError(result core.Result) *core.Err {
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok && err != nil {
		return &core.Err{Operation: "mcp.result", Message: "operation failed", Cause: err}
	}
	if result.Value != nil {
		return &core.Err{Operation: "mcp.result", Message: core.Sprint(result.Value)}
	}
	return &core.Err{Operation: "mcp.result", Message: result.Error()}
}
