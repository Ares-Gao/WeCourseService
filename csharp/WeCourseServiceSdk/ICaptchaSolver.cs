namespace WeCourseServiceSdk;

public interface ICaptchaSolver
{
    string Solve(byte[] imageBytes);
}
