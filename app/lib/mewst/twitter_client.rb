# typed: strict
# frozen_string_literal: true

class Mewst::TwitterClient
  extend T::Sig

  class Me < T::Struct
    const :id, String
    const :username, String
  end

  def initialize(access_token:)
    @access_token = access_token
  end

  def me
    @me ||= begin
      response = conn.get("/2/users/me")
      body = Oj.load(response.body)

      Me.new(
        id: body.dig("data", "id"),
        username: body.dig("data", "username")
      )
    end
  end

  private

  def conn
    @conn ||= Faraday.new(url: "https://api.twitter.com") do |f|
      f.request(:authorization, "Bearer", @access_token)
    end
  end
end
