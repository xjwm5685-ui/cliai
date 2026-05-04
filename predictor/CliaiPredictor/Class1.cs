using System.Diagnostics;
using System.Management.Automation;
using System.Management.Automation.Subsystem;
using System.Management.Automation.Subsystem.Prediction;
using System.Text;
using System.Text.Json;

namespace CliaiPredictor;

public sealed class CliaiCommandPredictor : ICommandPredictor
{
    private static readonly Guid PredictorId = new("5f593508-3fd9-4af3-a9d9-b0a76be8d259");
    private static readonly PredictorBridgeClient Bridge = new();
    private string? _lastInput;
    private string? _lastAcceptedInput;
    private string? _lastAcceptedSuggestion;

    public Guid Id => PredictorId;

    public string Name => "CliaiPredictor";

    public string Description => "Real-time command prediction powered by cliai.";

    public SuggestionPackage GetSuggestion(
        PredictionClient client,
        PredictionContext context,
        CancellationToken cancellationToken)
    {
        var input = context?.InputAst?.Extent?.Text ?? string.Empty;
        input = input.Trim();
        if (string.IsNullOrWhiteSpace(input))
        {
            return default;
        }

        if (TryGetCachedSuggestion(input, out var cachedSuggestion))
        {
            return BuildSuggestionPackage(cachedSuggestion);
        }

        var cwd = ResolveCurrentDirectory();
        var suggestions = Bridge.Query(input, cwd, 8, cancellationToken);
        if (suggestions.Count == 0)
        {
            return default;
        }

        _lastInput = input;
        _lastAcceptedInput = input;
        _lastAcceptedSuggestion = suggestions[0];
        return BuildSuggestionPackage(suggestions);
    }

    public bool CanAcceptFeedback(PredictionClient client, PredictorFeedbackKind feedback) => false;

    public void OnSuggestionDisplayed(PredictionClient client, uint session, int countOrIndex)
    {
    }

    public void OnSuggestionAccepted(PredictionClient client, uint session, string acceptedSuggestion)
    {
        _lastAcceptedSuggestion = acceptedSuggestion;
        if (!string.IsNullOrWhiteSpace(_lastInput))
        {
            _lastAcceptedInput = _lastInput;
        }
    }

    public void OnCommandLineAccepted(PredictionClient client, IReadOnlyList<string> history)
    {
    }

    public void OnCommandLineExecuted(PredictionClient client, string commandLine, bool success)
    {
    }

    private static SuggestionPackage BuildSuggestionPackage(IReadOnlyList<string> suggestions)
    {
        var items = new List<PredictiveSuggestion>(suggestions.Count);
        foreach (var suggestion in suggestions)
        {
            if (!string.IsNullOrWhiteSpace(suggestion) && !suggestion.Contains('\n') && !suggestion.Contains('\r'))
            {
                items.Add(new PredictiveSuggestion(suggestion));
            }
        }

        return items.Count == 0 ? default : new SuggestionPackage(items);
    }

    private bool TryGetCachedSuggestion(string input, out IReadOnlyList<string> suggestions)
    {
        if (!string.IsNullOrWhiteSpace(_lastAcceptedInput) &&
            !string.IsNullOrWhiteSpace(_lastAcceptedSuggestion) &&
            input.StartsWith(_lastAcceptedInput, StringComparison.OrdinalIgnoreCase) &&
            _lastAcceptedSuggestion.StartsWith(input, StringComparison.OrdinalIgnoreCase))
        {
            suggestions = new[] { _lastAcceptedSuggestion };
            return true;
        }

        suggestions = Array.Empty<string>();
        return false;
    }

    private static string ResolveCurrentDirectory()
    {
        return Environment.CurrentDirectory;
    }

    internal static void WarmUp() => Bridge.EnsureStarted();
}

public sealed class Init : IModuleAssemblyInitializer, IModuleAssemblyCleanup
{
    public void OnImport()
    {
        CliaiCommandPredictor.WarmUp();
        SubsystemManager.RegisterSubsystem(SubsystemKind.CommandPredictor, new CliaiCommandPredictor());
    }

    public void OnRemove(PSModuleInfo psModuleInfo)
    {
        SubsystemManager.UnregisterSubsystem(SubsystemKind.CommandPredictor, new Guid("5f593508-3fd9-4af3-a9d9-b0a76be8d259"));
    }
}

internal sealed class PredictorBridgeClient : IDisposable
{
    private static readonly TimeSpan ColdStartReadTimeout = TimeSpan.FromMilliseconds(250);
    private static readonly TimeSpan WarmReadTimeout = TimeSpan.FromMilliseconds(40);
    private static readonly TimeSpan ColdStartWindow = TimeSpan.FromSeconds(2);

    private readonly object _sync = new();
    private readonly JsonSerializerOptions _jsonOptions = new()
    {
        PropertyNameCaseInsensitive = true
    };

    private Process? _process;
    private StreamWriter? _stdin;
    private StreamReader? _stdout;
    private DateTime _startedAtUtc = DateTime.MinValue;
    private int _consecutiveTimeouts;

    public void EnsureStarted()
    {
        lock (_sync)
        {
            EnsureStartedLocked();
        }
    }

    public IReadOnlyList<string> Query(string input, string cwd, int limit, CancellationToken cancellationToken)
    {
        lock (_sync)
        {
            EnsureStartedLocked();
            if (_stdin is null || _stdout is null)
            {
                return Array.Empty<string>();
            }

            var request = JsonSerializer.Serialize(new PredictorBridgeRequest
            {
                Input = input,
                Cwd = cwd,
                Limit = limit
            });

            try
            {
                _stdin.WriteLine(request);
                _stdin.Flush();
            }
            catch
            {
                RestartLocked();
                return Array.Empty<string>();
            }

            try
            {
                var readTask = _stdout.ReadLineAsync();
                var completed = readTask.Wait(CurrentReadTimeout(), cancellationToken);
                if (!completed)
                {
                    _consecutiveTimeouts++;
                    if (_consecutiveTimeouts >= 2)
                    {
                        RestartLocked();
                    }
                    return Array.Empty<string>();
                }

                _consecutiveTimeouts = 0;
                var line = readTask.Result;
                if (string.IsNullOrWhiteSpace(line))
                {
                    return Array.Empty<string>();
                }

                var response = JsonSerializer.Deserialize<PredictorBridgeResponse>(line, _jsonOptions);
                if (response?.Suggestions is null || !string.IsNullOrWhiteSpace(response.Error))
                {
                    return Array.Empty<string>();
                }

                return response.Suggestions
                    .Select(item => item.Command)
                    .Where(item => !string.IsNullOrWhiteSpace(item))
                    .Distinct(StringComparer.OrdinalIgnoreCase)
                    .Take(limit)
                    .ToArray();
            }
            catch
            {
                RestartLocked();
                return Array.Empty<string>();
            }
        }
    }

    public void Dispose()
    {
        lock (_sync)
        {
            DisposeProcessLocked();
        }
    }

    private void EnsureStartedLocked()
    {
        if (_process is { HasExited: false } && _stdin is not null && _stdout is not null)
        {
            return;
        }

        DisposeProcessLocked();

        var process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = ResolveExecutable(),
                Arguments = "predictor serve --limit 8 --shell powershell",
                UseShellExecute = false,
                RedirectStandardInput = true,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                CreateNoWindow = true,
                StandardInputEncoding = Encoding.UTF8,
                StandardOutputEncoding = Encoding.UTF8
            }
        };

        process.Start();
        _process = process;
        _stdin = process.StandardInput;
        _stdout = process.StandardOutput;
        _startedAtUtc = DateTime.UtcNow;
        _consecutiveTimeouts = 0;
    }

    private void RestartLocked()
    {
        DisposeProcessLocked();
    }

    private void DisposeProcessLocked()
    {
        try
        {
            _stdin?.Dispose();
        }
        catch
        {
        }

        try
        {
            _stdout?.Dispose();
        }
        catch
        {
        }

        if (_process is not null)
        {
            try
            {
                if (!_process.HasExited)
                {
                    _process.Kill(true);
                }
            }
            catch
            {
            }

            _process.Dispose();
        }

        _process = null;
        _stdin = null;
        _stdout = null;
        _startedAtUtc = DateTime.MinValue;
        _consecutiveTimeouts = 0;
    }

    private TimeSpan CurrentReadTimeout()
    {
        if (_startedAtUtc != DateTime.MinValue && DateTime.UtcNow - _startedAtUtc <= ColdStartWindow)
        {
            return ColdStartReadTimeout;
        }

        return WarmReadTimeout;
    }

    private static string ResolveExecutable()
    {
        var configured = Environment.GetEnvironmentVariable("CLIAI_EXE");
        if (!string.IsNullOrWhiteSpace(configured))
        {
            return configured;
        }

        return OperatingSystem.IsWindows() ? "cliai.exe" : "cliai";
    }

    private sealed class PredictorBridgeRequest
    {
        public string Input { get; set; } = string.Empty;

        public string Cwd { get; set; } = string.Empty;

        public int Limit { get; set; }
    }

    private sealed class PredictorBridgeResponse
    {
        public List<PredictorCandidate>? Suggestions { get; set; }

        public string? Error { get; set; }
    }

    private sealed class PredictorCandidate
    {
        public string Command { get; set; } = string.Empty;
    }
}
