package com.company.modern;

import org.apache.ibatis.annotations.Mapper;
import com.company.common.ProductDTO;
import java.util.List;

/**
 * 상품 매퍼 인터페이스
 */
@Mapper
public interface ProductMapper {

    /**
     * 상품 등록
     */
    void insertProduct(ProductDTO product);

    /**
     * 전체 상품 조회
     */
    List<ProductDTO> selectAllProducts();

    /**
     * 키워드로 상품 검색
     */
    List<ProductDTO> selectProductsByKeyword(String keyword);
}
