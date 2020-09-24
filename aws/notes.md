Dev:

- Moved xword.club domain to AWS (just the nameservers; GoDaddy is still the registrar)
- Made an `xword.club` and a `www.xword.club` hosted zones
- Upload `buckets-and-user.yaml` as new cloudformation stack
- Create cert for `xword.club` and `www.xword.club` manually in ACM. No matter how many times I try I can't do it with goddamn Cloudformation. I think it's because I have them as two different hosted zones. This freezes forever when verifying the `www` subdomain, even though the DNS record that it wants me to add manually is ALREADY THERE. I add them anyway and it tells me to wait, and then after some time it pretends it never added them and asks me again. CFN SUCKS. I also tried it with one single hosted zone and it still doesn't want to do it. Maybe I need to transfer the domain to AWS proper. Fuck if I know.

```
Resources:
  Certificate:
    Type: "AWS::CertificateManager::Certificate"
    Properties:
      DomainName: !Ref DomainName
      DomainValidationOptions:
        - DomainName: !Ref DomainName
          HostedZoneId: !Ref HostedZoneId
        - DomainName: !Sub "www.${DomainName}"
          HostedZoneId: !Ref WWWHostedZoneId
      ValidationMethod: DNS
      SubjectAlternativeNames:
        - !Sub "www.${DomainName}"
```

For anyone who's better at AWS my ears are open.

- Upload `cloudfront.yaml`. Copy cert Arn created in last step.
  - Domain name is `xword.club`
  - OriginDomainName is `xword.club.s3.amazonaws.com` (the bucket + .s3.amazonaws.com)
  - OriginRedirectDomainName is also `xword.club.s3.amazonaws.com` (xxx maybe www. before?)
- Manually add CNAMEs to the created cloudfront distribution in Route53
  - this can probably be automated later.
