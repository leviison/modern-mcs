# Legacy Inventory Snapshot

Source reviewed: `../mcs`

## Runtime

- Shell installers and service scripts: `../mcs_configure.sh`, `../mcs/PartTwo.sh`
- Java binaries: `../mcs/mcs.jar`, `../mcs/java.jar`
- Native libs and plugin packages in `../mcs/plugins`

## Web/UI assets

- Main templates and scripts: `../mcs/www/example_templates`
- Report templates: `../mcs/www/reports`, `../mcs/publishtemplates`
- System/public pages: `../mcs/www/syspub`, `../mcs/data/syspub`

## Priority migration candidates

1. Security/auth flows (login, password reset, OTP, ACL settings)
2. SQL profile management and database export workflows
3. Report publication and scheduling flows
4. Satellite/deployment management pages
