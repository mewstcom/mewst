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
    form = Forms::EmailConfirmation.new(email:, locale:)
    result = Commands::SendEmailConfirmationCode.new(form:).call

    form = Forms::EmailConfirmationChallenge.new(
      email_confirmation_id: result.email_confirmation.id,
      confirmation_code: result.email_confirmation.code
    )
    Commands::ConfirmEmail.new(form:).call

    form = Forms::SignUp.new(atname:, email:, locale:, password:)
    result = Commands::SignUp.new(form:).call

    if avatar_url.present?
      result.profile.update!(avatar_url:)
    end
  end
end
