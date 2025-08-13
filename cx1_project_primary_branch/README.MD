This script provides a way to set the primary branch on projects that do not have a primary branch set but have run scans in the past.

There are four methods to execute this process - you can find all parameters by running "go run . -h" or compiling the code and running the executable with the -h flag.

1. By providing a list of projectIDs in an input file, with one project ID per line, and passing it to the script via -projects projectfile.txt
2. By providing an application name, so only projects inside the application are processed
3. By providing no input, so all projects in the system are processed.

In each of these 3 cases, the logic will proceed as follows:
1. The projects in scope are fetched by API
2. If a project does not have the primary branch set, the list of branches for the project are fetched by API
3. If there is only one branch active (previously scanned) in the project, that branch is set as primary. If there were no scans, or multiple branches were scanned, no change is made and the project is skipped.

This process includes multiple API calls to retrieve data from the system, which causes some performance overhead. For systems with large numbers of projects (eg: 30k+) and large numbers of scans per project, this overhead can be significant, resulting in slight system performance impacts and extended script runtime. To address performance concerns, there is a fourth method to invoke the script:
4. By providing a list of "projectId,branchName" pairs in an input file, with one pair per line, passed into the script via -branches projectBranches.txt

This allows to split the data-gathering and data-updating steps, and allows better control of the impact on the system. 

This "projectId,branchName" data can be retrieved via the database (if you have database access) using the following SQL script example:

```
create temp table projects_without_mainbranch as
  select id, '' as main_branch from projects where main_branch = '';

-- main, master, and then first scanned branch

update projects_without_mainbranch set main_branch = coalesce((select branch from scans where scans.project_id = projects_without_mainbranch.id and scans.branch ilike 'main' limit 1),'') where main_branch = '';

update projects_without_mainbranch set main_branch = coalesce((select branch from scans where scans.project_id = projects_without_mainbranch.id and scans.branch ilike 'master' limit 1),'') where main_branch = '';

update projects_without_mainbranch set main_branch = coalesce((select branch from scans where scans.project_id = projects_without_mainbranch.id order by created_at asc limit 1),'') where main_branch = '';

select * from projects_without_mainbranch where main_branch <>'';
```

This SQL will cause the process to set the primary branch to "main" if the project has a valid "main" branch, "master" if the project has a valid "master" branch, or otherwise it will use the branch specified in the first scan of the project. Additional branch names can be added in order of preference to the SQL statement above. The resulting data should be exported to a file - you may need to remove a header row and any double-quotes inserted by the SQL query tool that you use.