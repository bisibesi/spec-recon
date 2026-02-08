package com.company.legacy;

import org.apache.ibatis.annotations.Mapper;
import java.util.List;
import java.util.Map;

/**
 * 사용자 매퍼 인터페이스
 */
@Mapper
public interface UserMapper {

    /**
     * 인증 정보로 사용자 조회
     * 
     * @param param userId, password
     * @return 사용자 정보
     */
    Map<String, Object> selectUserByCredentials(Map<String, Object> param);

    /**
     * 전체 사용자 목록
     * 
     * @return 사용자 목록
     */
    List<Map<String, Object>> selectAllUsers();
}
