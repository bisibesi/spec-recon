package com.company.common;

/**
 * 문자열 유틸리티 클래스 (User-Defined Utility)
 * 
 * @author spec-recon-team
 */
public class StringUtil {

    /**
     * 문자열이 비어있는지 확인
     * 
     * @param str 검사할 문자열
     * @return 비어있으면 true
     */
    public static boolean isEmpty(String str) {
        return str == null || str.trim().length() == 0;
    }

    /**
     * 문자열이 비어있지 않은지 확인
     * 
     * @param str 검사할 문자열
     * @return 비어있지 않으면 true
     */
    public static boolean isNotEmpty(String str) {
        return !isEmpty(str);
    }

    /**
     * Null을 빈 문자열로 변환
     * 
     * @param str 변환할 문자열
     * @return 안전한 문자열
     */
    public static String nullToEmpty(String str) {
        return str == null ? "" : str;
    }
}
