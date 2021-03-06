# Provenance

Data versioning enables Pachyderm users to go back in time and see the state
of a dataset or repository at a particular moment in time. Data provenance
(from the French *provenir*, which means *the place of origin*),
also known as data lineage, tracks the dependencies and relationships
*between* datasets. Provenance answers not only the question of
where the data comes from, but also how the data was transformed along
the way. Data scientists use provenance in root cause analysis to improve
their code, workflows, and understanding of the data and its implications
on final results. Data scientists need
to have confidence in the information with which they operate. They need
to be able to reproduce the results and sometimes go through the whole
data transformation process from scratch multiple times, which makes data
provenance one of the most critical aspects of data analysis. If your
computations result in unexpected numbers, the first place to look
is the historical data that gives insights into possible flaws in the
transformation chain or the data itself.

For example, when a bank decides on a mortgage
application, many factors are taken into consideration, including the
credit history, annual income, and loan size. This data goes through multiple
automated steps of analysis with numerous dependencies and decisions made
along the way. If the final decision does not satisfy the applicant,
the historical data is the first place to look for proof of authenticity,
as well as for possible prejudice or model bias against the applicant.
Data provenance creates a complete audit trail that enables data scientists
to track the data from its origin through to the final decision and make
appropriate changes that address issues. With the adoption of the General Data
Protection Regulation (GDPR) compliance requirements, monitoring data lineage
is becoming a necessity for many organizations that work with sensitive data.

Pachyderm implements provenance for both commits and repositories.
You can track revisions of the data and
understand the connection between the data stored in one repository
and the results in the other
repository.

Collaboration takes data provenance even further. Provenance enables teams
of data scientists across the globe to build on each other work, share,
transform, and update datasets while automatically maintaining a
complete audit trail so that all results are reproducible.

The following diagram demonstrates how provenance works:

![Provenance example](../../images/provenance.svg)

In the diagram above, you can see two input repositories called `params`
and `data`. The `data` repository continuously collects
data from an outside source. The training model pipeline combines the
data from the first repository with the parameters in the second repository,
runs them through the pipeline code, and collects the results in the
output repo.

Provenance helps you to understand where commits in the output repo
originates in.
For example, in the diagram above, you can see that the commit `3b` was
created from the commit `1b` from the `data` repository and the commit `2a`
in the `params` repository. Similar, the commit `3a` was created from
the commit `1a` from the `data` repository and the commit `2a` from
the `params` repository.

## Tracking the Provenance Upstream

Pachyderm provides the `pachctl inspect commit` command that enables you to track
the provenance of your commits and learn where the data in the repository
originated.

**Example:**

```bash
$ pachctl inspect commit split@master
Commit: split@f71e42704b734598a89c02026c8f7d13
Original Branch: master
Started: 4 minutes ago
Finished: 3 minutes ago
Size: 0B
Provenance:  __spec__@8c6440f52a2d4aa3980163e25557b4a1 (split)  raw_data@ccf82debb4b94ca3bfe165aca8d517c3 (master)
```

In the example above, you can see that the latest commit in the master
branch of the split repository tracks back to the master branch in the
`raw_data` repository. The `__spec__` provenance shows you which
version of your code was run on the input commit
`ccf82debb4b94ca3bfe165aca8d517c3` in the `raw_data` repository to produce
the output commit `f71e42704b734598a89c02026c8f7d13` in the `split` repository.

## Tracking the Provenance Downstream

Pachyderm provides the `flush commit` command that enables you
to track the provenance downstream. Tracking downstream means that instead of
tracking the origin of a commit, you can learn in which output repository
a particular input has resulted.

For example, you have the `ccf82debb4b94ca3bfe165aca8d517c3` commit in
the `raw_data` repository. If you run the `pachctl flush commit` command
for this commit, you can see in which repositories and commits that data
resulted.

```bash
$ pachctl flush commit raw_data@ccf82debb4b94ca3bfe165aca8d517c3
REPO        BRANCH COMMIT                           PARENT STARTED        DURATION       SIZE
split       master f71e42704b734598a89c02026c8f7d13 <none> 52 minutes ago About a minute 25B
split       stats  9b46d7abf9a74bf7bf66c77f2a0da4b1 <none> 52 minutes ago About a minute 15.39MiB
pre_process master a99ab362dc944b108fb33544b2b24a8c <none> 48 minutes ago About a minute 100B
```

