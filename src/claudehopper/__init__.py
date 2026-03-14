"""claudehopper - Switch between Claude Code configuration profiles."""

from importlib.metadata import version, PackageNotFoundError

try:
    __version__ = version("claudehopper")
except PackageNotFoundError:
    __version__ = "dev"
