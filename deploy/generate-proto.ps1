$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$protoRoot = Join-Path $repoRoot "logic-grpc-service/proto"
$protos = @(Get-ChildItem -LiteralPath $protoRoot -Filter *.proto | Sort-Object Name)

foreach ($tool in @("protoc", "protoc-gen-go", "protoc-gen-go-grpc")) {
    Get-Command $tool -ErrorAction Stop | Out-Null
}

# One schema source generates both services; only the Go import paths differ.
foreach ($service in @("logic-grpc-service", "web-gin-service")) {
    $output = Join-Path $repoRoot "$service/pb"
    $arguments = @(
        "--proto_path=$protoRoot",
        "--go_out=$output", "--go_opt=paths=source_relative",
        "--go-grpc_out=$output", "--go-grpc_opt=paths=source_relative"
    )
    foreach ($proto in $protos) {
        $arguments += "--go_opt=M$($proto.Name)=$service/pb"
        $arguments += "--go-grpc_opt=M$($proto.Name)=$service/pb"
    }
    $arguments += $protos.FullName
    & protoc @arguments
    if ($LASTEXITCODE -ne 0) {
        throw "protoc failed for $service (exit $LASTEXITCODE)"
    }
}
