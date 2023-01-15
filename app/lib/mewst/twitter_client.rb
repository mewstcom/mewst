# typed: strict
# frozen_string_literal: true

class Mewst::TwitterClient
  extend T::Sig

  class Me < T::Struct
    const :id, String
    const :username, String
  end

  sig { params(access_token: String).void }
  def initialize(access_token:)
    @access_token = T.let(access_token, String)
  end

  sig { returns(Me) }
  def me
    @me ||= T.let(begin
      response = conn.get("/2/users/me")
      body = Oj.load(response.body)

      Me.new(
        id: body.dig("data", "id"),
        username: body.dig("data", "username")
      )
    end, T.nilable(Me))
  end

  private

  sig { returns(Faraday::Connection) }
  def conn
    @conn ||= T.let(Faraday.new(url: "https://api.twitter.com") do |f|
      f.request(:authorization, "Bearer", @access_token)
    end, T.nilable(Faraday::Connection))
  end
end
