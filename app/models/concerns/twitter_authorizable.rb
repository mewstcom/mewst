# typed: true
# frozen_string_literal: true

module TwitterAuthorizable
  extend T::Sig
  extend ActiveSupport::Concern

  def twitter_oauth2
    @twitter_oauth2 ||= Mewst::TwitterOauth2.new(
      client_id: ENV.fetch("MEWST_TWITTER_CLIENT_ID"),
      client_secret: ENV.fetch("MEWST_TWITTER_CLIENT_SECRET"),
      redirect_uri: ENV.fetch("MEWST_TWITTER_REDIRECT_URI")
    )
  end
end
