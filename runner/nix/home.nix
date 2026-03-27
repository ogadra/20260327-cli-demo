{ pkgs, ... }:
{
  home.username = "root";
  home.homeDirectory = "/root";
  home.stateVersion = "25.11";
  home.enableNixpkgsReleaseCheck = false;
  
  home.sessionVariables = {
    TZ = "Asia/Tokyo";
    LANG = "ja_JP.UTF-8";
    LC_ALL = "ja_JP.UTF-8";
    LOCALE_ARCHIVE = "${pkgs.glibcLocales}/lib/locale/locale-archive";
    TZDIR = "${pkgs.tzdata}/share/zoneinfo";
  };

  programs.bash.enable = true;
}
