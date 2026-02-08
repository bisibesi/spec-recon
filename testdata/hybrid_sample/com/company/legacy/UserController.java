package com.company.legacy;

import org.springframework.stereotype.Controller;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.servlet.ModelAndView;
import javax.servlet.http.HttpServletRequest;
import com.company.common.StringUtil;

/**
 * 사용자 관리 컨트롤러 (Legacy Spring MVC)
 * @author spec-recon-team
 */
@Controller
@RequestMapping("/user")
public class UserController {
    
    @Autowired
    private UserService userService;
    
    /**
     * 로그인 처리 (Legacy)
     * @param request HTTP 요청
     * @return ModelAndView
     */
    @RequestMapping(value = "/login", method = RequestMethod.POST)
    public ModelAndView login(HttpServletRequest request) {
        String userId = request.getParameter("userId");
        String password = request.getParameter("password");
        
        ModelAndView mav = new ModelAndView();
        
        if (StringUtil.isEmpty(userId) || StringUtil.isEmpty(password)) {
            mav.setViewName("error/400");
            return mav;
        }
        
        // Call service layer
        Map<String, Object> userInfo = userService.authenticateUser(userId, password);
        
        if (userInfo != null) {
            mav.addObject("user", userInfo);
            mav.setViewName("user/home");
        } else {
            mav.setViewName("user/login");
        }
        
        return mav;
    }
    
    /**
     * 사용자 목록 조회
     * @return ModelAndView
     */
    @RequestMapping(value = "/list", method = RequestMethod.GET)
    public ModelAndView getUserList() {
        ModelAndView mav = new ModelAndView();
        List<Map<String, Object>> users = userService.getAllUsers();
        mav.addObject("users", users);
        mav.setViewName("user/list");
        return mav;
    }
}
