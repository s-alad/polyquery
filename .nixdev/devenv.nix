{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.nodejs_latest
    pkgs.nodePackages.typescript
  ];

  # https://devenv.sh/languages/
  # languages.rust.enable = true;
  languages.javascript.enable = true;
  languages.javascript.package = pkgs.nodejs_latest;
  # languages.javascript.corepack.enable = true;  # enables yarn/pnpm (broken with nodejs_latest)
  languages.typescript.enable = true;
  languages.go.enable = true;

  # https://devenv.sh/processes/
  # processes.dev.exec = "${lib.getExe pkgs.watchexec} -n -- ls -la";

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/scripts/
  scripts.pqb.exec = "make build";
  scripts.pq-build-scout.exec = "make build-scout";
  scripts.pq-build-resolver.exec = "make build-resolver";
  scripts.pqc.exec = "make clean";

  # https://devenv.sh/basics/
  enterShell = ''
    echo " ▸ devenv online ◆ node $(node --version) · tsc $(tsc --version | cut -d' ' -f2) · go $(go version | cut -d' ' -f3 | sed 's/go//') · git $(git --version | cut -d' ' -f3)"
  '';

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
