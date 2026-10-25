// Package harness 是 suite 锚点包：仓内尚无业务代码时保持 ./... 非空，
// 使 go vet / go build / go test / go test -json 在空仓上可判定（ktor 脚手架
// 自带 Application.kt 同款先例）。首个业务包落地后本包可保留或删除。
package harness
