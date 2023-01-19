# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  CONTENT_MAXIMUM_LENGTH = 500

  belongs_to :profile

  validates :content, length: {maximum: CONTENT_MAXIMUM_LENGTH}, presence: true

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(String) }
  def timeline_item_score
    created_at!.to_datetime.strftime("%Q")
  end

  sig { returns(ActiveSupport::TimeWithZone) }
  def created_at!
    T.cast(created_at, ActiveSupport::TimeWithZone)
  end

  sig { returns(String) }
  def tweet_text
    max_tweet_length = 130

    if content.length < max_tweet_length
      return content
    end

    sliced_content = content.slice(0, max_tweet_length - 3)
    reversed_sliced_content = sliced_content.reverse
    url_match_regex = %r{\A[^ ]+\/\/:(ptth|sptth)}

    text = if reversed_sliced_content.match?(url_match_regex)
      reversed_sliced_content.sub(url_match_regex, "...").reverse
    else
      sliced_content + "..."
    end

    mewst_url = Rails.application.routes.url_helpers.post_url(profile!.atname, id, host: "www.mewst.com")

    [
      text,
      "",
      "Read more:",
      mewst_url
    ].join("\n")
  end
end
