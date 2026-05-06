# frozen_string_literal: true

OauthApplication.where(uid: OauthApplication::MEWST_WEB_UID).first_or_create!(
  name: "Mewst for Web",
  redirect_uri: "https://example.com/callback",
  scopes: ""
)
