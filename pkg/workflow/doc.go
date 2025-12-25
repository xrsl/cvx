// Package workflow manages AI workflow prompts for cvx commands.
//
// # Embedded Defaults
//
// Default workflow prompts are embedded at compile time from the defaults/ directory:
//   - defaults/add.md    - Job extraction prompt for 'cvx add'
//   - defaults/advise.md - Match analysis prompt for 'cvx advise'
//   - defaults/build.md  - CV tailoring prompt for 'cvx build'
//
// These files are embedded using //go:embed directives in workflow.go.
//
// # Updating Defaults
//
// To update default prompts:
//  1. Edit the relevant file in pkg/workflow/defaults/
//  2. Rebuild the binary with 'make build'
//
// The embedded content will be automatically updated at compile time.
//
// # Runtime Customization
//
// Users can customize prompts by creating files in .cvx/workflows/:
//   - .cvx/workflows/add.md
//   - .cvx/workflows/advise.md
//   - .cvx/workflows/build.md
//
// Run 'cvx init -r' to reset workflows to the embedded defaults.
package workflow
