# frozen_string_literal: true

OauthApplication.where(uid: OauthApplication::MEWST_WEB_UID).first_or_create!(
  name: "Mewst for Web",
  redirect_uri: "#{ENV.fetch("MEWST_WEB_URL")}/callback",
  scopes: ""
)

[
  %w[me@shimba.co ja shimbaco password https://shimba.co/img/shimbaco.jpg],
  ["user1@example.com", "ja", "user1", "password", ""],
  ["user2@example.com", "ja", "user2", "password", ""],
  ["user3@example.com", "ja", "user3", "password", ""]
].each do |(email, locale, atname, password, avatar_url)|
  ActiveRecord::Base.transaction do
    form = Internal::EmailConfirmationForm.new(email:, locale:)
    input = CreateEmailConfirmationService::Input.from_internal_form(form:)
    result = CreateEmailConfirmationService.new.call(input:)

    form = Internal::EmailConfirmationChallengeForm.new(
      email_confirmation_id: result.email_confirmation.id,
      confirmation_code: result.email_confirmation.code
    )
    input = ConfirmEmailService::Input.from_internal_form(form:)
    ConfirmEmailService.new.call(input:)

    form = Internal::AccountForm.new(atname:, email:, locale:, password:)
    input = CreateAccountService::Input.from_internal_form(form:)
    result = CreateAccountService.new.call(input:)

    if avatar_url.present?
      result.profile.update!(avatar_url:)
    end
  end
end
