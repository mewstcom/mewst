# typed: strict
# frozen_string_literal: true

class Dropdowns::RepostDropdownComponent < ApplicationComponent
  sig { params(post: Post, reposted: T::Boolean).void }
  def initialize(post:, reposted: false)
    @post = post
    @reposted = reposted
  end

  private

  sig { returns(Post) }
  attr_reader :post

  sig { returns(T::Boolean) }
  attr_reader :reposted

  sig { returns(String) }
  def repost_dropdown_button_class_names
    helpers.class_names("btn btn-sm dropdown-toggle rounded-pill", "btn-outline-secondary" => reposted, "btn-secondary" => !reposted)
  end

  sig { returns(String) }
  def repost_icon_class_names
    helpers.class_names("bi bi-repeat", "c-repost-dropdown--reposted" => reposted)
  end

  sig { returns(String) }
  def repost_count_class_names
    helpers.class_names("ms-1", "c-repost-dropdown--reposted" => reposted)
  end
end
