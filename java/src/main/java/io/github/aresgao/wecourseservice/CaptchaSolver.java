package io.github.aresgao.wecourseservice;

public interface CaptchaSolver {
    String solve(byte[] imageBytes) throws Exception;
}
