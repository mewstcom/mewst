# frozen_string_literal: true

OauthApplication.where(uid: OauthApplication::MEWST_WEB_UID).first_or_create!(
  name: "Mewst for Web",
  redirect_uri: "https://example.com/callback",
  scopes: ""
)

[
  %w[me@shimba.co ja shimbaco password https://shimba.co/img/shimbaco.jpg],
  ["user1@example.com", "ja", "user1", "password", ""],
  ["user2@example.com", "ja", "user2", "password", ""],
  ["user3@example.com", "ja", "user3", "password", ""]
].each do |(email, locale, atname, password, avatar_url)|
  CreateSeedAccountUseCase.new.call(atname:, email:, locale:, password:, avatar_url:)
end
