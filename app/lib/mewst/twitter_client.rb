# typed: strict
# frozen_string_literal: true

class Mewst::TwitterClient
  extend T::Sig

  def initialize(access_token:)
    @access_token = access_token
  end

  def me
    binding.irb
    conn.get("/2/users/me")
    conn.get("/2/users/shimbaco/following")
  end

  private

  def conn
    Faraday.new(
      url: "https://api.twitter.com",
      headers: {"Authorization: Bearer" => @access_token}
    )
  end
end

