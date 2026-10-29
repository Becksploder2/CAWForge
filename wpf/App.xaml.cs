using System;
using System.IO;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Threading;

namespace CAWForge;

public partial class App : Application
{
    private static readonly object LogLock = new();

    public App()
    {
        DispatcherUnhandledException += OnDispatcherException;
        AppDomain.CurrentDomain.UnhandledException += (_, e) =>
            LogException("Fatal runtime exception", e.ExceptionObject as Exception);
        TaskScheduler.UnobservedTaskException += (_, e) =>
        {
            LogException("Unobserved background task", e.Exception);
            e.SetObserved();
        };
    }

    private void OnDispatcherException(object sender, DispatcherUnhandledExceptionEventArgs e)
    {
        LogException("UI exception", e.Exception);
        // Recover from ordinary managed UI errors; never conceal fatal process corruption.
        if (e.Exception is OutOfMemoryException or AccessViolationException)
            return;
        e.Handled = true;
        try
        {
            MessageBox.Show("CAWForge encountered an error but is still running. Details were saved in the diagnostic log.",
                "CAWForge error", MessageBoxButton.OK, MessageBoxImage.Warning);
        }
        catch { /* Do not throw again while reporting an error. */ }
    }

    private static void LogException(string source, Exception? exception)
    {
        try
        {
            var folder = Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
                "CAWForge");
            Directory.CreateDirectory(folder);
            lock (LogLock)
            {
                File.AppendAllText(Path.Combine(folder, "errors.log"),
                    $"[{DateTimeOffset.Now:O}] {source}{Environment.NewLine}{exception}{Environment.NewLine}{Environment.NewLine}");
            }
        }
        catch { /* Logging must never crash the application. */ }
    }
}
