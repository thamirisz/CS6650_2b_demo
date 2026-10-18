# Product API on AWS ECS (Elastic Container Service) with Terraform

This repo deploys the Product API (Go + Gin) to AWS (Amazon Web Services) ECS.
Terraform builds the Docker image, pushes it to ECR (Elastic Container Registry), and runs it on ECS.

## Folder layout

| Path | What it does |
|---|---|
| `src/` | Server code (`product-service.go`, `go.mod`, `go.sum`) and `Dockerfile` |
| `terraform/provider.tf` | Sets up the AWS and Docker providers |
| `terraform/variables.tf` | Values you can change (region, names, port `8080`) |
| `terraform/modules/` | One folder per part: `ecr`, `ecs`, `logging`, `network` |
| `terraform/main.tf` | Calls the modules and passes values between them |
| `terraform/outputs.tf` | Prints the cluster and service names after deploy |

## 1. What you need

- Docker Desktop (running)
- Terraform (on a Mac with an M chip, use the **ARM64** version; check with `terraform version`, it should say `darwin_arm64`)
- AWS CLI (Command Line Interface)
- AWS Learner's Lab credentials

## 2. Set AWS credentials

Copy them from Learner's Lab → **AWS Details**. They expire when the lab session ends.

```bash
aws configure                       # region: us-west-2
aws configure set aws_session_token <YOUR-SESSION-TOKEN>
```

## 3. Deploy

```bash
git clone https://github.com/<your-username>/CS6650_2b_demo.git
cd CS6650_2b_demo/terraform

# Mac only: tell Terraform where Docker Desktop is
export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock

terraform init
terraform apply -auto-approve
```

## 4. Get the public IP (Internet Protocol) address

Wait 1–2 minutes for the task to start, then run this in the `terraform` folder:

```bash
aws ec2 describe-network-interfaces \
--network-interface-ids $(
    aws ecs describe-tasks \
    --cluster $(terraform output -raw ecs_cluster_name) \
    --tasks $(
        aws ecs list-tasks \
        --cluster $(terraform output -raw ecs_cluster_name) \
        --service-name $(terraform output -raw ecs_service_name) \
        --query 'taskArns[0]' --output text
    ) \
    --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
    --output text
) \
--query 'NetworkInterfaces[0].Association.PublicIp' \
--output text
```

The IP changes every time the task restarts, so run this again after each deploy.

## 5. Send requests

Replace `<IP>` with the address from step 4. Products 1 and 2 exist at start.

| Code | Request |
|---|---|
| GET 200 | `curl -i http://<IP>:8080/v1/products/1` |
| GET 404 | `curl -i http://<IP>:8080/v1/products/999` |
| GET 500 | `curl -i -H "X-Simulate-Error: true" http://<IP>:8080/v1/products/1` |
| POST 204 | see below |
| POST 400 | see below |
| POST 404 | see below |
| POST 500 | `curl -i -X POST http://<IP>:8080/v1/products/1/details -H "X-Simulate-Error: true" -H "Content-Type: application/json" -d '{}'` |

POST 204 – details saved:
```bash
curl -i -X POST http://<IP>:8080/v1/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":1,"sku":"NEW-SKU-1","manufacturer":"Acme","category_id":456,"weight":0,"some_other_id":789}'
```

POST 400 – bad input (missing `manufacturer`):
```bash
curl -i -X POST http://<IP>:8080/v1/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":1,"sku":"A","category_id":1,"weight":1,"some_other_id":1}'
```

POST 404 – product not found:
```bash
curl -i -X POST http://<IP>:8080/v1/products/999/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":999,"sku":"A","manufacturer":"M","category_id":1,"weight":1,"some_other_id":1}'
```

## 6. Clean up

Delete everything so you don't use up lab credit:

```bash
terraform destroy -auto-approve
```

## Common problems

| Error | Fix |
|---|---|
| `timeout while waiting for plugin to start` | Terraform is the Intel (amd64) version on an M-chip Mac. Install the ARM64 version. |
| `Cannot connect to the Docker daemon at unix:///var/run/docker.sock` | Start Docker Desktop and run the `export DOCKER_HOST=...` line from step 3. |
| `exec format error` in ECS logs | Image was built for ARM. The `Dockerfile` already builds for `linux/amd64`; don't remove that. |
| `already exists` on apply | Old ECR/ECS resources with the same names are still there. Delete them or change names in `variables.tf`. |

## Where things are

| What | Where |
|---|---|
| Server code | `src/product-service.go` |
| Dockerfile | `src/Dockerfile` |
| Infrastructure (Terraform) | `terraform/` |
| Load tests (Locust) + screenshots | [`partIV/README.md`](partIV/README.md) |
| Observability (Prometheus + Grafana + logs) | [`partV/README.md`](partV/README.md) |
