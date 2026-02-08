# Spec Recon (Legacy Spring Analyzer) 🦅

**Spec Recon**은 레거시 스프링(Spring Framework/Boot) 프로젝트를 정적 분석하여, 
API 명세서(Excel, HTML)와 Swagger(OpenAPI) 문서를 자동으로 생성해주는 도구입니다.

## 🚀 Features
- **Legacy Support:** 오래된 XML 매퍼(MyBatis)와 Controller/Service 구조 완벽 분석.
- **Deep Inference:** `Map`, `Object` 반환 타입도 코드 역추적을 통해 필드명을 찾아냅니다.
- **Clean Export:** 불필요한 DTO/VO 클래스를 제외한 깔끔한 엑셀 명세서 생성.
- **Swagger Generation:** `openapi.json`을 생성하여 Swagger UI에서 즉시 확인 가능.

## 📦 How to Use
1. 실행 파일(`spec-recon.exe`)을 다운로드합니다.
2. 터미널(CMD)에서 분석할 프로젝트 경로를 지정하여 실행합니다.

```bash
# Windows
spec-recon.exe -path "C:\MyLegacyProject"

# Mac/Linux
./spec-recon -path "/home/user/legacy-project"