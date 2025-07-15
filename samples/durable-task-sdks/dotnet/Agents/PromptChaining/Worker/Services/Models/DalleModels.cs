namespace AgentChainingSample.Services;

/// <summary>
/// Model for image description
/// </summary>
public class ImageDescription
{
    public string Description { get; set; } = string.Empty;
    public string Prompt { get; set; } = string.Empty;
    public string Caption { get; set; } = string.Empty;
}

/// <summary>
/// Model for DALL-E API response
/// </summary>
public class DalleResponse
{
    public DalleImageData[] data { get; set; } = Array.Empty<DalleImageData>();
}

/// <summary>
/// Model for DALL-E image data
/// </summary>
public class DalleImageData
{
    public string url { get; set; } = string.Empty;
}
