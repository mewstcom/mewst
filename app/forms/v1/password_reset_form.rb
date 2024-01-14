# typed: strict
# frozen_string_literal: true

class V1::PasswordResetForm < V1::ApplicationForm
  include Form::PasswordValidatable

  attribute :email, :string
  attribute :password, :string

  validates :email, presence: true
end
