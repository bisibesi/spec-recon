package com.company.modern;

import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.http.HttpStatus;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import com.company.common.ProductDTO;
import java.util.List;

/**
 * 상품 API 컨트롤러 (Modern REST API)
 */
@RestController
@RequestMapping("/api/v1/product")
@Tag(name = "Product API", description = "상품 관리 API")
public class ProductApiController {

    @Autowired
    private ProductService productService;

    /**
     * 상품 등록
     */
    @Operation(summary = "상품등록", description = "새로운 상품을 등록합니다")
    @PostMapping("/register")
    public ResponseEntity<ProductDTO> registerProduct(@RequestBody ProductDTO product) {
        ProductDTO result = productService.createProduct(product);
        return ResponseEntity.status(HttpStatus.CREATED).body(result);
    }

    /**
     * 상품 목록 조회
     */
    @Operation(summary = "상품목록조회", description = "전체 상품 목록을 조회합니다")
    @GetMapping("/list")
    public ResponseEntity<List<ProductDTO>> getProductList() {
        List<ProductDTO> products = productService.getProductList();
        return ResponseEntity.ok(products);
    }

    /**
     * 상품 검색
     */
    @Operation(summary = "상품검색", description = "키워드로 상품을 검색합니다")
    @PostMapping("/search")
    public ResponseEntity<List<ProductDTO>> searchProducts(@RequestBody String keyword) {
        List<ProductDTO> products = productService.searchByKeyword(keyword);
        return ResponseEntity.ok(products);
    }
}
