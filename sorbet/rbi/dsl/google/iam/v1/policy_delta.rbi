# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Google::Iam::V1::PolicyDelta`.
# Please instead update this file by running `bin/tapioca dsl Google::Iam::V1::PolicyDelta`.

class Google::Iam::V1::PolicyDelta
  sig do
    params(
      audit_config_deltas: T.nilable(T.any(Google::Protobuf::RepeatedField[Google::Iam::V1::AuditConfigDelta], T::Array[Google::Iam::V1::AuditConfigDelta])),
      binding_deltas: T.nilable(T.any(Google::Protobuf::RepeatedField[Google::Iam::V1::BindingDelta], T::Array[Google::Iam::V1::BindingDelta]))
    ).void
  end
  def initialize(audit_config_deltas: Google::Protobuf::RepeatedField.new(:message, Google::Iam::V1::AuditConfigDelta), binding_deltas: Google::Protobuf::RepeatedField.new(:message, Google::Iam::V1::BindingDelta)); end

  sig { returns(Google::Protobuf::RepeatedField[Google::Iam::V1::AuditConfigDelta]) }
  def audit_config_deltas; end

  sig { params(value: Google::Protobuf::RepeatedField[Google::Iam::V1::AuditConfigDelta]).void }
  def audit_config_deltas=(value); end

  sig { returns(Google::Protobuf::RepeatedField[Google::Iam::V1::BindingDelta]) }
  def binding_deltas; end

  sig { params(value: Google::Protobuf::RepeatedField[Google::Iam::V1::BindingDelta]).void }
  def binding_deltas=(value); end

  sig { void }
  def clear_audit_config_deltas; end

  sig { void }
  def clear_binding_deltas; end
end