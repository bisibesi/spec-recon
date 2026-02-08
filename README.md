# 🦅 Spec Recon (Legacy Spring Analyzer)

**Spec Recon**은 문서가 소실되거나 관리되지 않은 오래된 **Legacy Spring Framework/Boot** 프로젝트를 정적 분석(Static Analysis)하여, 최신 포맷의 **API 명세서(Swagger, Excel, HTML)**로 자동 변환해주는 도구입니다.

복잡한 XML 매퍼(MyBatis/iBatis), 컨트롤러, 서비스 간의 호출 구조를 역추적하여 숨겨진 입출력 필드(`Map`, `Object`)까지 찾아냅니다.

---

## 🚀 Key Features (핵심 기능)

- **Deep Analysis (심층 분석):** 단순한 코드 파싱을 넘어, 비즈니스 로직 내의 `Map.put`이나 DTO 변환 과정을 추적하여 **실제 리턴되는 데이터 구조**를 밝혀냅니다.
- **Auto-Generated Docs:** 한 번의 실행으로 개발자, 기획자, 테스터가 필요한 모든 형태의 문서를 생성합니다.
- **Legacy Support:** `@Controller`, `@Service`, XML 기반 SQL Mapper 등 구형 스프링 패턴을 완벽하게 지원합니다.
- **Smart Filtering:** 불필요한 DTO/VO 파일은 명세서에서 제외하고, 핵심 로직 위주로 요약합니다.

---

## 📂 Output Artifacts (결과물)

프로그램 실행 시 `output/` 폴더에 다음과 같은 문서들이 자동으로 생성됩니다.

```text
/output
├── 📄 openapi.json        # [API Spec] Swagger UI / Postman Import용 (개발자용)
├── 📄 spec-report.html    # [API Spec] 웹 뷰어 및 PDF 변환용 (공유/배포용)
├── 📄 spec-report.xlsx    # [Program Spec] 프로그램 상세 명세서 (기획/분석가용)
└── 📄 spec-report.doc     # [Doc Spec] 워드 문서 형태의 명세서 (문서화 제출용)
```

|파일명|용도|설명|
|---|---|---|
|openapi.json|Swagger 연동|Swagger UI나 Postman에 즉시 import하여 API 테스트 가능|
|spec-report.html|PDF 변환|브라우저에서 열어 바로 인쇄(PDF 저장) 가능한 깔끔한 보고서|
|spec-report.xlsx|프로그램 명세|API 목록, 입출력 필드, 호출 구조가 엑셀로 정리된 상세 명세서|
|spec-report.doc|워드 문서|보고용/제출용으로 편집 가능한 Word 형식의 API 명세서|

🛠 How to Use (실행 방법)
Spec Recon은 다양한 환경에 맞춰 3가지 실행 모드를 지원합니다. 편한 방법을 선택하세요.

방법 1. Config 설정 후 실행 (권장)
가장 표준적인 방법입니다. config.yaml 파일에 프로젝트 경로를 설정해두면 매번 경로를 입력할 필요가 없습니다.

1. config.yaml 파일을 열어 target_path를 수정합니다.
```yaml
# config.yaml
project:
  name: "My Legacy System"
  target_path: "C:/Projects/OldSpringProject"  # 분석할 프로젝트 경로
```
2. spec-recon.exe를 더블 클릭하여 실행합니다.

방법 2. 터미널(CMD)에서 경로 지정 실행
일회성 분석이나 스크립트 자동화 시 유용합니다.

```bash
# Windows
spec-recon.exe -path "C:\Projects\TargetProject"

# Mac / Linux
./spec-recon -path "/home/user/projects/target-project"
```

방법 3. 프로젝트 Root에 복사 후 실행 (Drop-in)
설정 파일조차 귀찮을 때 사용합니다. 실행 파일을 분석할 프로젝트 폴더 안에 넣고 돌리면 됩니다.

1. spec-recon.exe 파일을 분석하려는 프로젝트의 **최상위 폴더(Root)**로 복사합니다.

2. 그 자리에서 실행합니다. (옵션이 없으면 현재 폴더를 자동으로 스캔합니다.)

⚙️ Configuration (config.yaml)
```yaml
project:
  name: "Legacy System Analysis"  # 리포트 제목으로 표시됨
  target_path: "./"               # 분석 대상 경로 (기본값: 현재 폴더)
  
analyzer:
  deep_scan: true                 # Map/Object 추론 로직 활성화 여부
  exclude_pattern:                # 분석에서 제외할 폴더/파일 패턴
    - ".git"
    - "test"
    - "node_modules"

output:
  format: ["json", "html", "excel", "doc"] # 생성할 문서 포맷 지정
```

🏗 Build & Install
Go 언어 환경이 설치되어 있다면 직접 빌드할 수 있습니다.
```bash
# 의존성 다운로드
go mod tidy

# 빌드 (Windows)
go build -o spec-recon.exe ./cmd/spec-recon

# 빌드 (Mac/Linux)
go build -o spec-recon ./cmd/spec-recon
```

🦅 Created by Spec Recon Team
"We bring light to the dark corners of legacy code."
