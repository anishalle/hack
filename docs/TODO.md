# TODO

- App support is intentionally hidden in the v1 CLI. The backend still models
  secrets as app + environment so multi-app workflows can be exposed later, but
  user-facing commands currently use the hidden app name `default` to keep the
  CLI intuitive while the tool is greenfield.
