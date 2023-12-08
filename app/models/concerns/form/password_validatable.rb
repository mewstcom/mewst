# typed: strict
# frozen_string_literal: true

module Form::PasswordValidatable
  extend ActiveSupport::Concern

  included do
    validates :password,
      length: {in: 8..ActiveModel::SecurePassword::MAX_PASSWORD_LENGTH_ALLOWED},
      presence: true
  end
end
