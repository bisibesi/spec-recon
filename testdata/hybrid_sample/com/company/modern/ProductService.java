package com.company.modern;

import org.springframework.stereotype.Service;
import org.springframework.beans.factory.annotation.Autowired;
import com.company.common.ProductDTO;
import com.company.common.StringUtil;
import java.util.List;

/**
 * 상품 서비스 (Modern)
 */
@Service
public class ProductService {

    @Autowired
    private ProductMapper productMapper;

    /**
     * 상품 등록
     */
    public ProductDTO createProduct(ProductDTO product) {
        if (StringUtil.isEmpty(product.getProductName())) {
            throw new IllegalArgumentException("상품명은 필수입니다");
        }
        productMapper.insertProduct(product);
        return product;
    }

    /**
     * 상품 목록 조회
     */
    public List<ProductDTO> getProductList() {
        return productMapper.selectAllProducts();
    }

    /**
     * 키워드로 상품 검색
     */
    public List<ProductDTO> searchByKeyword(String keyword) {
        return productMapper.selectProductsByKeyword(keyword);
    }
}
