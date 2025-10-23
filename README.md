# standard-template
A standard template for a blocky project

Set up some settings:
- [ ] General Settings: The "standard-template" contains a good starting point.
  See "Settings > General".
- [ ] Access control: A good getting started option is to set the group
  "blocky/admin" as admins and "blocky/developers" as maintainers.
- [ ] Rulesets: A good getting starting option can be found in the
  `standard-template` project under "Settings > Rules > Rulesets > Merge to
  main".  You can export that ruleset and import it into your new project.

Customize labels:
- [ ] Check out the labels in the `standard-template` for the common setup.
  Note that for the auto approve bot in the "On PRs" workflow looks for a label
  with name "auto-approve-me"

Customize a few files:
- [ ] Check the PR template in `.gihub/pull_request_template.md`.  It is very
  basic but will get the job done.
- [ ] Customize the "On PRs" workflow located at `.github/workflows/on-prs.yml`
