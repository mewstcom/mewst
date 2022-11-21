# typed: strict
# frozen_string_literal: true

class Authentication
  include ActiveModel::Model
  include ActiveModel::Attributes

  attribute :back, :string
  attribute :email, :string

  validates :back, format: {with: %r{\A/}, allow_blank: true}
  validates :email, email: true, presence: true

  def confirm_email_to_sign_up
    account = Account.find_by(email:)
    return if account

    email_confirmation = EmailConfirmation.create!(email_confirmation_attributes(event: :sign_up))
    EmailConfirmationMailer.sign_up_confirmation(email_confirmation.id, I18n.locale.to_s).deliver_later
  end

  def confirm_email_to_sign_in
    account = Account.find_by(email:)
    return unless account

    email_confirmation = EmailConfirmation.create!(email_confirmation_attributes(event: :sign_in))
    EmailConfirmationMailer.sign_in_confirmation(email_confirmation.id, I18n.locale.to_s).deliver_later
  end

  private

  def email_confirmation_attributes(event:, account: nil, back: nil)
    {
      account:,
      email:,
      event:,
      token: SecureRandom.uuid,
      back:,
      expires_at: Time.current + EmailConfirmation::EXPIRES_IN,
    }
  end
end
