package com.example.monitoring;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

/**
 * HTTP request interceptor for monitoring request metrics
 */
@Component
public class RequestInterceptor implements HandlerInterceptor {
    
    private static final Logger logger = LoggerFactory.getLogger(RequestInterceptor.class);
    
    @Autowired
    private SpringBootMonitor springBootMonitor;
    
    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, 
                           Object handler) throws Exception {
        request.setAttribute("startTime", System.currentTimeMillis());
        return true;
    }
    
    @Override
    public void afterCompletion(HttpServletRequest request, HttpServletResponse response, 
                              Object handler, Exception ex) throws Exception {
        
        long startTime = (Long) request.getAttribute("startTime");
        long responseTime = System.currentTimeMillis() - startTime;
        
        // Record request metrics
        springBootMonitor.recordRequest(responseTime);
        
        // Record errors
        if (response.getStatus() >= 400 || ex != null) {
            springBootMonitor.recordError();
        }
        
        // Log request details for analytics
        logger.info("Request completed - Method: {}, URI: {}, Status: {}, ResponseTime: {}ms", 
            request.getMethod(), 
            request.getRequestURI(), 
            response.getStatus(), 
            responseTime);
    }
}