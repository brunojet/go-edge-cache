# NOTE: ServiceNow Credentials are not created by this project.
# Secrets should be created and managed separately in the external project.
#
# resource "aws_secretsmanager_secret" "servicenow" {
#   name                    = "/go-edge-cache/servicenow"
#   recovery_window_in_days = 30  # Protege contra delecao acidental (30 dias recovery)
#
#   tags = var.tags
#
#   lifecycle {
#     prevent_destroy = true  # Impede delecao via terraform destroy
#   }
# }
