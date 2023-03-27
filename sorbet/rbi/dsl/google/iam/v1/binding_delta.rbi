# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Google::Iam::V1::BindingDelta`.
# Please instead update this file by running `bin/tapioca dsl Google::Iam::V1::BindingDelta`.

class Google::Iam::V1::BindingDelta
  sig do
    params(
      action: T.nilable(T.any(Symbol, Integer)),
      condition: T.nilable(Google::Type::Expr),
      member: T.nilable(String),
      role: T.nilable(String)
    ).void
  end
  def initialize(action: nil, condition: nil, member: nil, role: nil); end

  sig { returns(T.any(Symbol, Integer)) }
  def action; end

  sig { params(value: T.any(Symbol, Integer)).void }
  def action=(value); end

  sig { void }
  def clear_action; end

  sig { void }
  def clear_condition; end

  sig { void }
  def clear_member; end

  sig { void }
  def clear_role; end

  sig { returns(T.nilable(Google::Type::Expr)) }
  def condition; end

  sig { params(value: T.nilable(Google::Type::Expr)).void }
  def condition=(value); end

  sig { returns(String) }
  def member; end

  sig { params(value: String).void }
  def member=(value); end

  sig { returns(String) }
  def role; end

  sig { params(value: String).void }
  def role=(value); end
end