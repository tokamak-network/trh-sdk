# TODO

## Plan

- [x] Electron DRB E2E에서 확인된 `tokamak-deployer` cache dir 초기화 실패를 재현하고 root cause를 문서화
- [x] `ensureTokamakDeployer()`가 cache parent directory를 생성하는 failing test 추가
- [x] 최소 구현으로 test pass 및 관련 regression 검증
- [ ] backend image 재빌드 후 Electron DRB E2E 재검증
- [x] 현재 `trh-sdk`에서 execution client(`op-geth`) 결합 지점 식별
- [x] Sepolia에서 `ethrex` 교체를 위한 실행 가이드 문서 작성
- [x] README에 신규 가이드 링크 추가
- [x] 결과 검토 및 변경 파일 확인

## Review

- Electron DRB E2E 실행으로 `tokamak-deployer` binary cache 경로(`~/.trh/bin`)가 없는 컨테이너에서 첫 다운로드가 실패하는 runtime bug를 확인했다.
- `ensureTokamakDeployerWithVersion()`에서 cache dir를 먼저 `os.MkdirAll()` 하도록 수정했고, 누락된 디렉터리 생성 regression test를 추가했다.
- `GOTOOLCHAIN=auto go test ./pkg/stacks/thanos/...` 기준으로 관련 패키지 검증을 마쳤다.
- 신규 문서 `docs/sepolia-ethrex-migration-guide.md`를 추가해, Sepolia 배포 시 execution client를 `ethrex`로 교체하는 전략/변경 지점/검증 체크리스트를 정리했다.
- `README.md`에 해당 문서 링크를 추가해 접근성을 확보했다.
