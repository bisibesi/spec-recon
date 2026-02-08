package com.company.legacy;

import org.springframework.stereotype.Service;
import org.springframework.beans.factory.annotation.Autowired;
import java.util.List;
import java.util.Map;

/**
 * 사용자 서비스 (Legacy)
 */
@Service
public class UserService {
    
    @Autowired
    private UserMapper userMapper;
    
    /**
     * 사용자 인증
     * @param userId 사용자 ID
     * @param password 비밀번호
     * @return 사용자 정보
     */
    public Map<String, Object> authenticateUser(String userId, String password) {
        Map<String, Object> param = new HashMap<>();
        param.put("userId", userId);
        param.put("password", password);
        return userMapper.selectUserByCredentials(param);
    }
    
    /**
     * 전체 사용자 목록 조회
     * @return 사용자 목록
     */
    public List<Map<String, Object>> getAllUsers() {
        return userMapper.selectAllUsers();
    }
}
