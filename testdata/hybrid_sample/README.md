# Hybrid Sample Project

This is a test dataset for **Spec Recon** - a tool that analyzes Legacy Spring (Java/XML) codebases.

## 📁 Project Structure

```
testdata/hybrid_sample/
├── com/company/
│   ├── legacy/              # Legacy Spring MVC Pattern
│   │   ├── UserController.java
│   │   ├── UserService.java
│   │   └── UserMapper.java
│   ├── modern/              # Modern REST API Pattern
│   │   ├── ProductApiController.java
│   │   ├── ProductService.java
│   │   └── ProductMapper.java
│   └── common/              # Utilities & DTOs (Should be filtered)
│       ├── StringUtil.java
│       └── ProductDTO.java
└── resources/sqlmap/
    ├── UserMapper.xml       # MyBatis XML for UserMapper
    └── ProductMapper.xml    # MyBatis XML for ProductMapper
```

## 🎯 Test Scenarios

### 1️⃣ Legacy Call Chain (UserController → UserService → UserMapper → XML)
- **Entry Point:** `UserController.login()`
- **Service:** `UserService.authenticateUser()`
- **Mapper:** `UserMapper.selectUserByCredentials()`
- **SQL:** `UserMapper.xml` - `selectUserByCredentials`

### 2️⃣ Modern Call Chain (ProductApiController → ProductService → ProductMapper → XML)
- **Entry Point:** `ProductApiController.registerProduct()`
- **Service:** `ProductService.createProduct()`
- **Mapper:** `ProductMapper.insertProduct()`
- **SQL:** `ProductMapper.xml` - `insertProduct`

### 3️⃣ Utility Filter Test
- **StringUtil.java** - Should be filtered out (User-defined utility)
- **ProductDTO.java** - Should be filtered out (Plain object)

## 🧪 Expected Parser Behavior

### Controllers
- ✅ Detect `@Controller` (Legacy)
- ✅ Detect `@RestController` (Modern)
- ✅ Extract `@Autowired` dependencies
- ✅ Parse `@RequestMapping` / `@PostMapping` / `@GetMapping`

### Services
- ✅ Detect `@Service` annotation
- ✅ Extract `@Autowired` dependencies to Mappers

### Mappers
- ✅ Detect `@Mapper` annotation
- ✅ Extract interface methods
- ✅ Link to XML via `namespace + id`

### XML Files
- ✅ Parse `<mapper namespace="...">`
- ✅ Extract `<select>`, `<insert>`, `<update>`, `<delete>` with `id`
- ✅ Extract SQL queries

### Filters
- ❌ Exclude `StringUtil` (ends with "Util")
- ❌ Exclude `ProductDTO` (ends with "DTO")

## 📊 Expected Output Format (Hierarchical Tree)

```
Row 0: [CTRL] UserController.login()
Row 1:   [SVC] UserService.authenticateUser()
Row 2:     [MAP] UserMapper.selectUserByCredentials()
Row 3:       [SQL] SELECT user_id, user_name... FROM tb_user WHERE...

Row 4: [CTRL] ProductApiController.registerProduct()
Row 5:   [SVC] ProductService.createProduct()
Row 6:     [MAP] ProductMapper.insertProduct()
Row 7:       [SQL] INSERT INTO tb_product...
```

## 🔍 Verification Checklist

- [ ] Parser can read all Java files
- [ ] Parser can handle EUC-KR/UTF-8 encoding
- [ ] Linker connects Controller → Service via variable name
- [ ] Linker connects Service → Mapper via variable name
- [ ] Linker connects Mapper → XML via namespace + id
- [ ] Walker generates hierarchical DFS tree
- [ ] Excel generator creates proper indentation
- [ ] StringUtil and ProductDTO are excluded from output

---

**Note:** This sample uses Korean comments (로그인, 상품등록) to test EUC-KR encoding handling.
