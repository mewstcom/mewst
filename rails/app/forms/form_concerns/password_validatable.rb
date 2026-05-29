# typed: strict
# frozen_string_literal: true

module FormConcerns::PasswordValidatable
  extend ActiveSupport::Concern

  included do
    validates :password, length: {minimum: UserRecord::PASSWORD_MIN_LENGTH}, presence: true
    validate do |record|
      password_value = record.password

      if password_value.present? && password_value.bytesize > ActiveModel::SecurePassword::MAX_PASSWORD_LENGTH_ALLOWED
        record.errors.add(:password, :password_too_long)
      end
    end
  end
end
