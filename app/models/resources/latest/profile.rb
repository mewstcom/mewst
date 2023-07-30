# typed: false
# frozen_string_literal: true

class Resources::Latest::Profile < Resources::Latest::Base
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :name
end
